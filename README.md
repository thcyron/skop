# Skop: Simple Kubernetes Operators for Go

[![GoDoc](https://godoc.org/github.com/thcyron/skop?status.svg)](https://godoc.org/github.com/thcyron/skop)
![](https://github.com/thcyron/skop/workflows/CI/badge.svg)

**Skop** is a lightweight framework for writing Kubernetes operators in Go. It:

* Tries to keep the amount of boilerplate code small.
* Uses the minimal [k8s package](https://github.com/ericchiang/k8s) for communicating with the Kubernetes API.
* Doesn’t rely on code generation.
* Provides helpers for common reconciliation tasks.

## Usage

Basically, writing an operator for a custom resource boils down to:

1.  Defining the custom resource as a Go struct:

    ```go
    type Test struct {
        Kind       string             `json:"kind"`
        APIVersion string             `json:"apiVersion"`
        Metadata   *metav1.ObjectMeta `json:"metadata"`
        Spec       TestSpec           `json:"spec"`
    }

    type TestSpec struct {
        Text string `json:"text"`
    }
    ```

2.  Creating the operator:

    ```go
    op := skop.New(
        skop.WithResource(&Test{}),
        skop.WithReconciler(skop.ReconcilerFunc(reconciler)),
    )
    ```

3.  Writing the reconcile function:

    ```go
    func reconciler(ctx context.Context, op *skop.Operator, res k8s.Resource) error {
        test := res.(*Test)
        deployment := makeDeployment(test)
        return reconcile.Deployment(ctx, op.Client(), deployment)
    }
    ```

4.  Running the operator:

    ```go
    go op.Run()
    ```

A complete, working example can be found in the [example/](example/) directory.

## License

MIT
