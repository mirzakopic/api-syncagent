# CRD Puller

The `crd-puller` can be used for testing and development in order to export a
CustomResourceDefinition for any Group/Version/Kind (GVK) in a Kubernetes cluster.

The main difference between this and kcp's own `crd-puller` is that this one
works based on GVKs and not resources (i.e. on `apps/v1 Deployment` instead of
`apps.deployments`). This is more useful since a PublishedResource publishes a
specific Kind and version.

## Usage

```shell
export KUBECONFIG=/path/to/kubeconfig

./crd-puller Deployment.v1.apps.k8s.io
```
