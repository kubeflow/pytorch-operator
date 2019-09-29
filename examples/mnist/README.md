### Distributed MNIST Examples

This folder contains an example where mnist is trained. This example is also used for e2e testing.

The python script used to train mnist with pytorch takes in several arguments that can be used
to switch the distributed backends. The manifests to launch the distributed training of this mnist
file using the pytorch operator are under the respective version folders: [v1](./v1).
Each folder contains manifests with example usage of the different backends.

**Build Image**

The default image name and tag is `kubeflow/pytorch-dist-mnist-test:1.0`.

```shell
docker build -f Dockerfile -t kubeflow/pytorch-dist-mnist-test:1.0 ./
```
NOTE: If you you are working on Power System, Dockerfile.ppc64le could be used.

**Create the mnist PyTorch job**

The below example uses the gloo backend.

```shell
kubectl create -f ./v1beta1/pytorch_job_mnist_gloo.yaml
```
