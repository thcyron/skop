package skop

import (
	"context"
	"io"

	"github.com/ericchiang/k8s"
)

//go:generate mockgen -source=client.go -package mock -mock_names Client=Client,Watcher=Watcher -destination mock/client.go Client,Watcher

type Client interface {
	Create(ctx context.Context, res k8s.Resource, options ...k8s.Option) error
	Get(ctx context.Context, name string, res k8s.Resource, options ...k8s.Option) error
	Update(ctx context.Context, res k8s.Resource, options ...k8s.Option) error
	Delete(ctx context.Context, res k8s.Resource, options ...k8s.Option) error
	Watch(ctx context.Context, res k8s.Resource) (Watcher, error)
}

type Watcher interface {
	Next(res k8s.Resource) (string, error)
	Close() error
}

// FromK8sClient provides an implementation of the Client interface
// backed by the provided *k8s.Client.
func FromK8sClient(c *k8s.Client) Client {
	return &k8sClientAdapter{c}
}

type k8sClientAdapter struct {
	c *k8s.Client
}

func (a k8sClientAdapter) Create(ctx context.Context, res k8s.Resource, options ...k8s.Option) error {
	return a.c.Create(ctx, res, options...)
}

func (a k8sClientAdapter) Get(ctx context.Context, name string, res k8s.Resource, options ...k8s.Option) error {
	return a.c.Get(ctx, a.c.Namespace, name, res, options...)
}

func (a k8sClientAdapter) Update(ctx context.Context, res k8s.Resource, options ...k8s.Option) error {
	return a.c.Update(ctx, res, options...)
}

func (a k8sClientAdapter) Delete(ctx context.Context, res k8s.Resource, options ...k8s.Option) error {
	return a.c.Delete(ctx, res, options...)
}

func (a k8sClientAdapter) Watch(ctx context.Context, res k8s.Resource) (Watcher, error) {
	w, err := a.c.Watch(ctx, a.c.Namespace, res, k8s.ResourceVersion("0"))
	if err != nil {
		return nil, err
	}
	return k8sWatcherAdapter{w}, nil
}

type k8sWatcherAdapter struct {
	w *k8s.Watcher
}

func (a k8sWatcherAdapter) Next(res k8s.Resource) (event string, err error) {
	event, err = a.w.Next(res)
	// We need to be able to detect EOF errors, but unfortunately the k8s package
	// prefixes errors and thus erases the original error. Undo that by comparing
	// the error message.
	if err != nil && err.Error() == "decode event: EOF" {
		err = io.EOF
	}
	return
}

func (a k8sWatcherAdapter) Close() error { return a.w.Close() }
