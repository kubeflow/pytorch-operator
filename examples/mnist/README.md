### Distributed MNIST Examples

This folder contains an example where mnist is trained. This example is also used for e2e testing.

The python script used to train mnist with pytorch takes in several arguments that can be used
to switch the distributed backends. The manifests to launch the distributed training of this mnist
file using the pytorch operator are under the respective version folders: [v1beta1](./v1beta1) and [v1beta2](./v1beta2).
Each folder contains manifests with example usage of the different backends.

**Build Image**

The default image name and tag is `kubeflow/pytorch-dist-mnist-test:1.0`.

```shell
docker build -f Dockerfile -t kubeflow/pytorch-dist-mnist-test:1.0 ./
```

**Create the mnist PyTorch job**

The below example uses the gloo backend.

```shell
kubectl create -f ./v1beta1/pytorch_job_mnist_gloo.yaml
```
