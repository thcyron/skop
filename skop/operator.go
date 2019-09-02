package skop

import (
	"context"
	"fmt"
	"io"
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/ericchiang/k8s"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type Operator struct {
	client         Client
	resourceType   reflect.Type
	logger         log.Logger
	reconciler     Reconciler
	store          store
	updates        chan k8s.Resource
	retries        chan string
	retrySchedules map[string]retrySchedule
	stop           chan struct{}
	stopOnce       sync.Once
}

type retrySchedule struct {
	timer    *time.Timer
	failures uint
}

type Option func(op *Operator)

// WithResource configures an operator to watch for changes of the specified
// resource type. This option is required and New will panic if it is not provided.
func WithResource(r k8s.Resource) Option {
	return func(op *Operator) {
		op.resourceType = reflect.TypeOf(r).Elem()
	}
}

// WithClient configures an operator to use the specified client to communicate
// with the Kubernetes API. This option accepts an *k8s.Client as well as anything
// implementing the Client interface and panics for any other values. It is
// required and New panics if it is not provided.
func WithClient(client interface{}) Option {
	return func(op *Operator) {
		switch c := client.(type) {
		case *k8s.Client:
			op.client = &k8sClientAdapter{c}
		case Client:
			op.client = c
		default:
			panic(fmt.Sprintf("skop: unsupported client type: %T", c))
		}
	}
}

// WithLogger configures an operator to use the specified logger. This option is
// optional and defaults to using the standard library's log package.
func WithLogger(logger log.Logger) Option {
	return func(op *Operator) {
		op.logger = logger
	}
}

// WithReconciler configures the operator to use the specified reconciler. As an operator can
// only have one reconciler, when specifying this option multiple times, the last option wins.
func WithReconciler(r Reconciler) Option {
	return func(op *Operator) {
		op.reconciler = r
	}
}

// New constructs a new operator with the provided options.
func New(options ...Option) *Operator {
	op := &Operator{
		updates:        make(chan k8s.Resource),
		retries:        make(chan string),
		stop:           make(chan struct{}),
		retrySchedules: make(map[string]retrySchedule),
	}
	for _, option := range options {
		option(op)
	}
	if op.resourceType == nil {
		panic("skop: no resource configured")
	}
	if op.client == nil {
		panic("skop: no client configured")
	}
	if op.reconciler == nil {
		panic("skop: no reconciler configured")
	}
	if op.logger == nil {
		op.logger = log.NewLogfmtLogger(log.StdlibWriter{})
	}
	return op
}

func (op *Operator) makeResource() k8s.Resource {
	return reflect.New(op.resourceType).Interface().(k8s.Resource)
}

func (op *Operator) Run() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		op.watch()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		op.reconcile()
		wg.Done()
	}()

	wg.Wait()
}

func (op *Operator) Stop() {
	op.stopOnce.Do(func() {
		close(op.stop)
	})
}

func (op *Operator) watch() {
	var (
		ctx = context.Background()
		res = op.makeResource()
	)

	for {
		op.store.Clear()

		watchCtx, cancel := context.WithCancel(ctx)
		watchErr := make(chan error, 1)

		go func() {
			level.Info(op.logger).Log(
				"msg", "starting to watch for changes",
			)
			watcher, err := op.client.Watch(watchCtx, res)
			if err != nil {
				watchErr <- err
				return
			}
			defer watcher.Close()

			for {
				res = op.makeResource()
				eventType, err := watcher.Next(res)
				if err != nil {
					if err == io.EOF {
						err = nil
					}
					watchErr <- err
					return
				}
				name := res.GetMetadata().GetName()
				switch eventType {
				case k8s.EventAdded, k8s.EventModified:
					level.Info(op.logger).Log(
						"msg", "adding resource to store",
						"resource", name,
					)
					op.store.Add(res)
					op.updates <- res
				case k8s.EventDeleted:
					level.Info(op.logger).Log(
						"msg", "removing resource from store",
						"resource", name,
					)
					op.store.Remove(res)
				default:
					break
				}
			}
		}()

		select {
		case <-op.stop:
			cancel()
			return
		case err := <-watchErr:
			cancel()
			if err != nil {
				level.Error(op.logger).Log(
					"msg", "watch failed",
					"err", err,
				)
			} else {
				level.Debug(op.logger).Log(
					"msg", "watcher stopped without error",
				)
			}
		}
	}
}

func (op *Operator) reconcile() {
	for {
		level.Debug(op.logger).Log(
			"msg", "waiting for update or retry",
		)
		var res k8s.Resource
		select {
		case <-op.stop:
			return
		case res = <-op.updates:
			level.Debug(op.logger).Log(
				"msg", "got resource from update channel",
				"resource", res.GetMetadata().GetName(),
			)
		case name := <-op.retries:
			level.Debug(op.logger).Log(
				"msg", "got resource from retries channel",
				"resource", name,
			)
			res = op.store.Get(name)
		}
		if res == nil {
			continue
		}
		ctx := context.Background()
		level.Debug(op.logger).Log(
			"msg", "calling reconciler",
			"resource", res.GetMetadata().GetName(),
		)
		start := time.Now()
		op.runReconciler(ctx, res)
		level.Debug(op.logger).Log(
			"msg", "reconciler finished",
			"resource", res.GetMetadata().GetName(),
			"duration", time.Since(start),
		)
	}
}

const maxBackoff = 5 * time.Minute

func (op *Operator) runReconciler(ctx context.Context, res k8s.Resource) {
	defer func() {
		if r := recover(); r != nil {
			level.Error(op.logger).Log(
				"msg", "reconciler panicked",
				"reason", r,
			)
		}
	}()

	name := res.GetMetadata().GetName()

	if schedule, ok := op.retrySchedules[name]; ok {
		schedule.timer.Stop()
	}

	err := op.reconciler.Reconcile(ContextWithLogger(ctx, op.logger), op, res)
	if err == nil {
		level.Debug(op.logger).Log(
			"msg", "reconciler ran without errors; removing scheduled retry",
			"resource", name,
		)
		delete(op.retrySchedules, name)
		return
	}

	failures := op.retrySchedules[name].failures
	backoff := time.Duration(math.Pow(2, float64(failures))) * time.Second
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	level.Debug(op.logger).Log(
		"msg", "reconciler failed; scheduling retry",
		"resource", name,
		"backoff", backoff,
	)
	timer := time.AfterFunc(backoff, func() {
		select {
		case <-op.stop:
			return
		case op.retries <- name:
		}
	})
	op.retrySchedules[name] = retrySchedule{
		failures: failures + 1,
		timer:    timer,
	}
}

// Client returns a client for the Kubernetes API you can use in your handlers.
func (op *Operator) Client() Client {
	return op.client
}

// Reconcile reconciles all currently known resources.
func (op *Operator) Reconcile() {
	level.Info(op.logger).Log(
		"msg", "reconciling all resources",
	)
	go func() {
		for _, res := range op.store.All() {
			select {
			case <-op.stop:
				return
			case op.retries <- res.GetMetadata().GetName():
				level.Debug(op.logger).Log(
					"msg", "triggered update",
					"resource", res.GetMetadata().GetName(),
				)
			}
		}
		level.Debug(op.logger).Log(
			"msg", "triggered update for all resources",
		)
	}()
}
