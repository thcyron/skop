package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/thcyron/skop/v2/reconcile"
	"github.com/thcyron/skop/v2/skop"
)

type Test struct {
	metav1.ObjectMeta `json:"metadata"`
	Kind              string   `json:"kind"`
	APIVersion        string   `json:"apiVersion"`
	Spec              TestSpec `json:"spec"`
}

type TestSpec struct {
	Text string `json:"text"`
}

func main() {
	logger := makeLogger()

	config, err := makeConfig()
	if err != nil {
		level.Error(logger).Log(
			"msg", "failed to create config",
			"err", err,
		)
		os.Exit(1)
	}

	op := skop.New(
		skop.WithResource("example.com", "v1", "tests", &Test{}),
		skop.WithConfig(config),
		skop.WithReconciler(skop.ReconcilerFunc(reconciler)),
		skop.WithLogger(logger),
	)

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- op.Run()
	}()

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		level.Info(logger).Log(
			"msg", "received signal",
			"signal", sig,
		)
		op.Stop()
	case err := <-runErrCh:
		level.Error(logger).Log(
			"msg", "operator failed",
			"err", err,
		)
		os.Exit(1)
	}
}

func reconciler(ctx context.Context, op *skop.Operator, res skop.Resource) error {
	test := res.(*Test)

	skop.Logger(ctx).Log(
		"msg", "handler called",
		"resource", test.Name,
	)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: test.ObjectMeta.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					Kind:       test.Kind,
					APIVersion: test.APIVersion,
					Name:       test.ObjectMeta.Name,
					UID:        test.ObjectMeta.UID,
					Controller: skop.Bool(true),
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"deployment": "test",
				},
			},
			Replicas: skop.Int32(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deployment": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:  "test",
							Image: "alpine:3.10",
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

	return reconcile.Deployment(ctx, op.Clientset(), deployment)
}

func makeLogger() log.Logger {
	var logger log.Logger
	logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	return logger
}

func makeConfig() (*rest.Config, error) {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
