package skop_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ericchiang/k8s"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/golang/mock/gomock"

	"github.com/thcyron/skop/skop"
	"github.com/thcyron/skop/skop/mock"
)

func init() {
	k8s.Register("example.com", "v1", "test", true, &testResource{})
}

type testResource struct {
	Metadata *metav1.ObjectMeta `json:"metadata"`
}

func (r *testResource) GetMetadata() *metav1.ObjectMeta { return r.Metadata }

func generation(i int64) *int64 { return &i }

func TestOperator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock.NewClient(ctrl)

	var (
		reconcilerFuncs   = make(chan func() error)
		reconcilerResults = make(chan error)
		reconciler        = func(ctx context.Context, op *skop.Operator, res k8s.Resource) error {
			err := (<-reconcilerFuncs)()
			reconcilerResults <- err
			return err
		}
	)

	var (
		watcherFuncs = make(chan func(k8s.Resource) (string, error))
		watcher      = mock.NewWatcher(ctrl)
	)
	watcher.
		EXPECT().
		Next(gomock.Any()).
		DoAndReturn(func(res k8s.Resource) (string, error) {
			return (<-watcherFuncs)(res)
		}).
		AnyTimes()
	watcher.
		EXPECT().
		Close().
		Return(nil).
		AnyTimes()

	client.
		EXPECT().
		Watch(gomock.Any(), gomock.Eq("test"), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns string, res k8s.Resource) (skop.Watcher, error) {
			return watcher, nil
		}).
		AnyTimes()

	op := skop.New(
		skop.WithResource(&testResource{}),
		skop.WithNamespace("test"),
		skop.WithClient(client),
		skop.WithReconciler(skop.ReconcilerFunc(reconciler)),
	)

	runExited := make(chan struct{})
	go func() {
		op.Run()
		close(runExited)
	}()

	// Emit an ADDED event.
	watcherFuncs <- func(res k8s.Resource) (string, error) {
		res.(*testResource).Metadata = &metav1.ObjectMeta{
			Name:       k8s.String("test"),
			Namespace:  k8s.String("skop"),
			Generation: generation(1),
		}
		return k8s.EventAdded, nil
	}

	// Expect the reconciler to be called.
	reconcilerFuncs <- func() error { return nil }
	if err := <-reconcilerResults; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Emit a MODIFIED event.
	watcherFuncs <- func(res k8s.Resource) (string, error) {
		res.(*testResource).Metadata = &metav1.ObjectMeta{
			Name:       k8s.String("test"),
			Namespace:  k8s.String("skop"),
			Generation: generation(2),
		}
		return k8s.EventModified, nil
	}

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

	// Make the watcher fail.
	watcherFuncs <- func(res k8s.Resource) (string, error) {
		return "", errors.New("boom")
	}

	// Emit another MODIFIED event. Operator should have created a new watcher.
	watcherFuncs <- func(res k8s.Resource) (string, error) {
		res.(*testResource).Metadata = &metav1.ObjectMeta{
			Name:       k8s.String("test"),
			Namespace:  k8s.String("skop"),
			Generation: generation(3),
		}
		return k8s.EventModified, nil
	}
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

	// Emit a DELETED event.
	watcherFuncs <- func(res k8s.Resource) (string, error) {
		res.(*testResource).Metadata = &metav1.ObjectMeta{
			Name:       k8s.String("test"),
			Namespace:  k8s.String("skop"),
			Generation: generation(3),
		}
		return k8s.EventDeleted, nil
	}

	// Stop the operator.
	op.Stop()
	<-runExited
}
