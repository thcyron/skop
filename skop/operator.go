package skop

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Operator struct {
	namespace      string
	config         *rest.Config
	clientset      *kubernetes.Clientset
	resource       schema.GroupVersionResource
	resourceType   reflect.Type
	informer       informer
	logger         log.Logger
	reconciler     Reconciler
	updates        chan Resource
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

// WithNamespace configures an operator to only watch for resources
// in the specified namespace. By default, or when an empty namespace
// is specified, the operator watches for resources in all namespaces.
func WithNamespace(namespace string) Option {
	return func(op *Operator) {
		op.namespace = namespace
	}
}

// WithResource tells an operator the API group, version, and resource
// name (plural) and also specifies a prototype of the Go struct which
// represents the custom resource.
func WithResource(group, version, resource string, prototype Resource) Option {
	return func(op *Operator) {
		op.resource = schema.GroupVersionResource{
			Group:    group,
			Version:  version,
			Resource: resource,
		}
		op.resourceType = reflect.TypeOf(prototype).Elem()
	}
}

// WithConfig configures an operator to use the specified config
// for creating Kubernetes clients.
func WithConfig(c *rest.Config) Option {
	return func(op *Operator) {
		op.config = c
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
		updates:        make(chan Resource),
		retries:        make(chan string),
		stop:           make(chan struct{}),
		retrySchedules: make(map[string]retrySchedule),
	}
	for _, option := range options {
		option(op)
	}
	if op.resource.Resource == "" {
		panic("skop: no resource configured")
	}
	if op.config == nil {
		panic("skop: no config configured")
	}
	if op.reconciler == nil {
		panic("skop: no reconciler configured")
	}
	if op.logger == nil {
		op.logger = log.NewLogfmtLogger(log.StdlibWriter{})
	}
	return op
}

func (op *Operator) Run() error {
	if op.informer == nil {
		informer, err := newK8sInformer(op.config, op.namespace, op.resource, op.resourceType)
		if err != nil {
			return err
		}
		op.informer = informer
	}

	cs, err := kubernetes.NewForConfig(op.config)
	if err != nil {
		return err
	}
	op.clientset = cs

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
	return nil
}

func (op *Operator) Stop() {
	op.stopOnce.Do(func() {
		close(op.stop)
	})
}

func (op *Operator) watch() {
	level.Info(op.logger).Log("msg", "starting informer")
	op.informer.Run(op.stop, func(res Resource) {
		op.updates <- res
	})
}

func (op *Operator) reconcile() {
	for {
		level.Debug(op.logger).Log(
			"msg", "waiting for update or retry",
		)
		var res Resource
		select {
		case <-op.stop:
			return
		case res = <-op.updates:
			level.Debug(op.logger).Log(
				"msg", "got resource from update channel",
				"resource", res.GetName(),
			)
		case key := <-op.retries:
			level.Debug(op.logger).Log(
				"msg", "got resource from retries channel",
				"resource", key,
			)
			if r := op.informer.Get(key); r != nil {
				res = r
			} else {
				level.Debug(op.logger).Log(
					"msg", "informer did not return resource",
					"resource", key,
				)
			}
		}
		if res == nil {
			continue
		}
		ctx := context.Background()
		level.Debug(op.logger).Log(
			"msg", "calling reconciler",
			"resource", res.GetName(),
		)
		start := time.Now()
		op.runReconciler(ctx, res)
		level.Debug(op.logger).Log(
			"msg", "reconciler finished",
			"resource", res.GetName(),
			"duration", time.Since(start),
		)
	}
}

const maxBackoff = 5 * time.Minute

func (op *Operator) runReconciler(ctx context.Context, res Resource) {
	defer func() {
		if r := recover(); r != nil {
			level.Error(op.logger).Log(
				"msg", "reconciler panicked",
				"reason", r,
			)
		}
	}()

	key := op.informer.Key(res)

	if schedule, ok := op.retrySchedules[key]; ok {
		schedule.timer.Stop()
	}

	err := op.reconciler.Reconcile(ContextWithLogger(ctx, op.logger), op, res)
	if err == nil {
		level.Debug(op.logger).Log(
			"msg", "reconciler ran without errors; removing scheduled retry",
			"resource", key,
		)
		delete(op.retrySchedules, key)
		return
	}

	failures := op.retrySchedules[key].failures
	backoff := time.Duration(math.Pow(2, float64(failures))) * time.Second
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	level.Debug(op.logger).Log(
		"msg", "reconciler failed; scheduling retry",
		"resource", key,
		"backoff", backoff,
		"err", err,
	)

	timer := time.AfterFunc(backoff, func() {
		select {
		case <-op.stop:
			return
		case op.retries <- key:
		}
	})
	op.retrySchedules[key] = retrySchedule{
		failures: failures + 1,
		timer:    timer,
	}
}

// Reconcile reconciles all currently known resources.
func (op *Operator) Reconcile() {
	level.Info(op.logger).Log(
		"msg", "reconciling all resources",
	)
	go func() {
		for _, key := range op.informer.Keys() {
			select {
			case <-op.stop:
				return
			case op.retries <- key:
				level.Debug(op.logger).Log(
					"msg", "triggered update",
					"resource", key,
				)
			}
		}
		level.Debug(op.logger).Log(
			"msg", "triggered update for all resources",
		)
	}()
}

func (op *Operator) Config() *rest.Config {
	return op.config
}

func (op *Operator) Clientset() *kubernetes.Clientset {
	return op.clientset
}

func (op *Operator) UpdateStatus(ctx context.Context, res Resource) error {
	obj := &unstructured.Unstructured{}
	data, err := json.Marshal(res)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &obj.Object); err != nil {
		return err
	}
	client, err := dynamic.NewForConfig(op.config)
	if err != nil {
		return err
	}
	_, err = client.
		Resource(op.resource).
		Namespace(res.GetNamespace()).
		UpdateStatus(ctx, obj, metav1.UpdateOptions{})
	return err
}
