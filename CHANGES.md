# Changes

## v2.1.0

- Add option to set default resync interval

## v2.0.1

- Use background delete propagation when reconciling job, daemon set, and
  deployment absence

## v2.0.0

- Ditch github.com/ericchiang/k8s in favor of k8s.io/client-go
- Import path is now github.com/thcyron/skop/v2

## v1.2.0

- Add reconcile.DaemonSet()

## v1.1.0

- Add FromK8sClient() to get an implementation of the Client interface backed
  by a *k8s.Client

## v1.0.0

- Initial release
