# Developer Guide

PyTorch-operator is currently at v1.

## Building the operator

```sh
cd ${GOPATH}/src/github.com/kubeflow
git clone git@github.com:kubeflow/pytorch-operator.git
```

Resolve dependencies (if you don't have dep install, check how to do it [here](https://github.com/golang/dep))

Install dependencies

```sh
dep ensure
```

Build it

```sh
go install github.com/kubeflow/pytorch-operator/cmd/pytorch-operator.v1
```

## Running the Operator Locally

Running the operator locally (as opposed to deploying it on a K8s cluster) is convenient for debugging/development.

### Run a Kubernetes cluster

First, you need to run a Kubernetes cluster locally. There are lots of choices:

- [local-up-cluster.sh in Kubernetes](https://github.com/kubernetes/kubernetes/blob/master/hack/local-up-cluster.sh)
- [minikube](https://github.com/kubernetes/minikube)

`local-up-cluster.sh` runs a single-node Kubernetes cluster locally, but Minikube runs a single-node Kubernetes cluster inside a VM. It is all compilable with the controller, but the Kubernetes version should be `1.8` or above.

Notice: If you use `local-up-cluster.sh`, please make sure that the kube-dns is up, see [kubernetes/kubernetes#47739](https://github.com/kubernetes/kubernetes/issues/47739) for more details.

### Configure KUBECONFIG and KUBEFLOW_NAMESPACE

We can configure the operator to run locally using the configuration available in your kubeconfig to communicate with
a K8s cluster. Set your environment:

```sh
export KUBECONFIG=$(echo ~/.kube/config)
export KUBEFLOW_NAMESPACE=$(your_namespace)
```

* KUBEFLOW_NAMESPACE is used when deployed on Kubernetes, we use this variable to create other resources (e.g. the resource lock) internal in the same namespace. It is optional, use `default` namespace if not set.

### Create the PyTorch Operator CRD

After the cluster is up, the PyTorch Operator CRD should be created on the cluster.

```bash
kubectl create -f ./manifests/crd.yaml
```

### Run Operator

Now we are ready to run operator locally:

```sh
pytorch-operator.v1
```

To verify local operator is working, create an example job and you should see jobs created by it.

```sh
cd ./examples/mnist
docker build -f Dockerfile -t kubeflow/pytorch-dist-mnist-test:1.0 ./
kubectl create -f ./v1/pytorch_job_mnist_gloo.yaml
```

## Go version

On ubuntu the default go package appears to be gccgo-go which has problems see [issue](https://github.com/golang/go/issues/15429) golang-go package is also really old so install from golang tarballs instead.
