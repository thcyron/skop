package skop

import (
	"context"
	"errors"
	"sync"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type testResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
}

type testInformer struct {
	mu        sync.Mutex
	updates   chan Resource
	resources map[string]Resource
}

func newTestInformer() *testInformer {
	return &testInformer{
		updates:   make(chan Resource),
		resources: map[string]Resource{},
	}
}

func (i *testInformer) Get(key string) Resource {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.resources[key]
}

func (i *testInformer) Key(res Resource) string {
	return res.GetName()
}

func (i *testInformer) Keys() []string {
	all := []string{}
	i.mu.Lock()
	for key := range i.resources {
		all = append(all, key)
	}
	i.mu.Unlock()
	return all
}

func (i *testInformer) Run(stopCh <-chan struct{}, update func(Resource)) {
	for {
		select {
		case res := <-i.updates:
			update(res)
		case <-stopCh:
			return
		}
	}
}

func (i *testInformer) add(res Resource) {
	i.mu.Lock()
	i.resources[res.GetName()] = res
	i.mu.Unlock()
	i.updates <- res
}

func TestOperator(t *testing.T) {
	var (
		reconcilerFuncs   = make(chan func() error)
		reconcilerResults = make(chan error)
		reconciler        = func(ctx context.Context, op *Operator, res Resource) error {
			err := (<-reconcilerFuncs)()
			reconcilerResults <- err
			return err
		}
	)

	op := New(
		WithResource("example.com", "v1", "tests", &testResource{}),
		WithConfig(&rest.Config{}),
		WithReconciler(ReconcilerFunc(reconciler)),
	)

	informer := newTestInformer()
	op.informer = informer

	runExited := make(chan struct{})
	go func() {
		op.Run()
		close(runExited)
	}()

	// Add a new resource.
	informer.add(&testResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test",
			Namespace:  "skop",
			Generation: 1,
		},
	})

	// Expect the reconciler to be called.
	reconcilerFuncs <- func() error { return nil }
	if err := <-reconcilerResults; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Update that resource.
	informer.add(&testResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test",
			Namespace:  "skop",
			Generation: 2,
		},
	})

	// Let the reconciler fail and the operator schedule a retry.
	boom := errors.New("boom")
	reconcilerFuncs <- func() error { return boom }
	if err := <-reconcilerResults; err != boom {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for operator to call the reconciler again.
	reconcilerFuncs <- func() error { return nil }
	if err := <-reconcilerResults; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Trigger a manual reconcile run.
	op.Reconcile()
	reconcilerFuncs <- func() error { return nil }
	if err := <-reconcilerResults; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Stop the operator.
	op.Stop()
	<-runExited
}
