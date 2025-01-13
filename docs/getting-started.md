# Getting Started with the Sync Agent

All that is necessary to run the Sync Agent is a running Kubernetes cluster (for testing you can use
[kind][kind]) [kcp][kcp] installation.

## Prerequisites

- A running Kubernetes cluster to run the Sync Agent in.
- A running kcp installation as the source of truth.
- A kubeconfig with admin or comparable permissions in a specific kcp workspace.

## APIExport Setup

Before installing the Sync Agent it is necessary to create an `APIExport` on kcp. The `APIExport` should
be empty, because it is updated later by the Sync Agent, but it defines the new API group we're
introducing. An example file could look like this:

```yaml
apiVersion: apis.kcp.io/v1alpha1
kind: APIExport
metadata:
  name: test.example.com
spec: {}
```

Create a file with a similar content (you most likely want to change the name, as that is the API
group under which your published resources will be made available) and create it in a kcp workspace
of your choice:

```sh
# use the kcp kubeconfig
$ export KUBECONFIG=/path/to/kcp.kubeconfig

# nativagate to the workspace wher the APIExport should exist
$ kubectl ws :workspace:you:want:to:create:it

# create it
$ kubectl create --filename apiexport.yaml
apiexport/test.example.com created
```

## Sync Agent Installation

The Sync Agent can be installed into any namespace, but in our example we are going with `k8c-system`.
It doesn't necessarily have to live in the same Kubernetes cluster where it is synchronizing data
to, but that is the common setup. Ultimately the Sync Agent synchronizes data between two kube
endpoints.

Now that the `APIExport` is created, switch to the Kubernetes cluster from which you wish to
[publish resources](publish-resources.md). You will need to ensure that a kubeconfig with access to
the kcp workspace that the `APIExport` has been created in is stored as a `Secret` on this cluster.
Make sure that the kubeconfig points to the right workspace (not necessarily the `root` workspace).

This can be done via a command like this:

```sh
$ kubectl create secret generic kcp-kubeconfig \
  --namespace k8c-system \
  --from-file "kubeconfig=admin.kubeconfig"
```

The Sync Agent is shipped as a Helm chart and to install it, the next step is preparing a `values.yaml`
file for the Sync Agent Helm chart. We need to pass the target `APIExport`, a name for the Sync Agent
itself and a reference to the kubeconfig secret we just created.

```yaml
syncAgent:
  # Required: the name of the APIExport in kcp that this Sync Agent is supposed to serve.
  apiExportName: test.example.com

  # Required: this Sync Agent's public name, will be shown in kcp, purely for informational purposes.
  agentName: unique-test

  # Required: Name of the Kubernetes Secret that contains a "kubeconfig" key, with the kubeconfig
  # provided by kcp to access it.
  kcpKubeconfig: kcp-kubeconfig

  # Create additional RBAC on the service cluster. These rules depend somewhat on the Sync Agent
  # configuration, but the following two rules are very common. If you configure the Sync Agent to
  # only work with cluster-scoped objects, you do not need to grant it permissions to create
  # namespaces, for example.
  rbac:
    createClusterRole: true
    rules:
      # in order to create APIResourceSchemas
      - apiGroups:
          - apiextensions.k8s.io
        resources:
          - customresourcedefinitions
        verbs:
          - get
          - list
          - watch
      # so copies of remote objects can be placed in their target namespaces
      - apiGroups:
          - ""
        resources:
          - namespaces
        verbs:
          - get
          - list
          - watch
          - create
```

In addition, it is important to create RBAC rules for the resources you want to publish. If you want
to publish the `Certificate` resource as created by cert-manager, you will need to append the
following ruleset:

```yaml
      # so we can manage certificates
      - apiGroups:
          - cert-manager.io
        resources:
          - certificates
        verbs:
          - '*'
```

Once this `values.yaml` file is prepared, install a recent development build of the Sync Agent:

```sh
helm install kcp-api-syncagent oci://github.com/kcp-dev/helm-charts/api-syncagent --version 9.9.9-9fc9a430d95f95f4b2210f91ef67b3ec153b5cab -f values.yaml -n kcp-system
```

Two `kcp-api-syncagent` Pods should start in the `kcp-system` namespace. If they crash you will need to
identify the reason from container logs. A possible issue is that the provided kubeconfig does not
have permissions against the target kcp workspace.

## Publish Resources

Once the Sync Agent Pods are up and running, you should be able to follow the
[Publishing Resources](publish-resources.md) guide.

## Consume Service

Once resources have been published through the Sync Agent, they can be consumed on the kcp side (i.e.
objects on kcp will be synced back and forth with the service cluster). Follow the
guide to [consuming services](consuming-services.md).

[kind]: https://github.com/kubernetes-sigs/kind
[kcp]: https://kcp.io
