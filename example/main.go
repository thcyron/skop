package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ericchiang/k8s"
	appsv1 "github.com/ericchiang/k8s/apis/apps/v1"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/ghodss/yaml"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	isatty "github.com/mattn/go-isatty"

	"github.com/thcyron/skop/skop"
	"github.com/thcyron/skop/reconcile"
)

func init() {
	k8s.Register("example.com", "v1", "tests", true, &Test{})
}

type Test struct {
	Kind       string             `json:"kind"`
	APIVersion string             `json:"apiVersion"`
	Metadata   *metav1.ObjectMeta `json:"metadata"`
	Spec       TestSpec           `json:"spec"`
}

func (t *Test) GetMetadata() *metav1.ObjectMeta { return t.Metadata }

type TestSpec struct {
	Text string `json:"text"`
}

func main() {
	logger := makeLogger()

	client, err := makeClient()
	if err != nil {
		level.Error(logger).Log(
			"msg", "failed to create client",
			"err", err,
		)
		os.Exit(1)
	}

	op := skop.New(
		skop.WithResource(&Test{}),
		skop.WithClient(client),
		skop.WithNamespace(client.Namespace),
		skop.WithReconciler(skop.ReconcilerFunc(reconciler)),
		skop.WithLogger(logger),
	)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		op.Run()
		wg.Done()
	}()

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	level.Info(logger).Log(
		"msg", "received signal",
		"signal", sig,
	)

	op.Stop()
	wg.Wait()
}

func reconciler(ctx context.Context, op *skop.Operator, res k8s.Resource) error {
	test := res.(*Test)

	skop.Logger(ctx).Log(
		"msg", "handler called",
		"resource", test.GetMetadata().GetName(),
	)

	deployment := &appsv1.Deployment{
		Metadata: &metav1.ObjectMeta{
			Name:      k8s.String("test"),
			Namespace: test.GetMetadata().Namespace,
			OwnerReferences: []*metav1.OwnerReference{
				&metav1.OwnerReference{
					Kind:       k8s.String(test.Kind),
					ApiVersion: k8s.String(test.APIVersion),
					Name:       test.Metadata.Name,
					Uid:        test.Metadata.Uid,
					Controller: k8s.Bool(true),
				},
			},
		},
		Spec: &appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"deployment": "test",
				},
			},
			Replicas: k8s.Int32(1),
			Template: &corev1.PodTemplateSpec{
				Metadata: &metav1.ObjectMeta{
					Labels: map[string]string{
						"deployment": "test",
					},
				},
				Spec: &corev1.PodSpec{
					Containers: []*corev1.Container{
						&corev1.Container{
							Name:  k8s.String("test"),
							Image: k8s.String("alpine:3.10"),
							Args: []string{
								"sh",
								"-c",
								fmt.Sprintf("while true; do echo %s; sleep 1; done", test.Spec.Text),
							},
						},
					},
				},
			},
		},
	}

	return reconcile.Deployment(ctx, op.Client(), deployment)
}

func makeLogger() log.Logger {
	w := log.NewSyncWriter(os.Stdout)
	var logger log.Logger
	if isatty.IsTerminal(os.Stdout.Fd()) {
		logger = log.NewLogfmtLogger(w)
	} else {
		logger = log.NewJSONLogger(w)
	}
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	return logger
}

func makeKubeconfigClient(path string) (*k8s.Client, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	config := new(k8s.Config)
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}
	client, err := k8s.NewClient(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func makeClient() (*k8s.Client, error) {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return makeKubeconfigClient(kubeconfig)
	}
	return k8s.NewInClusterClient()
}
