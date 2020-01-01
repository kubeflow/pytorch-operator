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
The above command will fail in non-gcp enviromnets as it expects an uset to have a gcp user and image path is gcr.io/<your_project>/pytorch_dist_mnist:latest

For Testing it on non gcp cluster like on-premises edit the file ./v1/pytorch_job_mnist_gloo.yaml replace the image name gcr.io/<your_project>/pytorch_dist_mnist:latest with kubeflow/pytorch-dist-mnist-test:1.0

Example yaml with local image:
```
apiVersion: "kubeflow.org/v1"
kind: "PyTorchJob"
metadata:
  name: "pytorch-dist-mnist-gloo"
spec:
  pytorchReplicaSpecs:
    Master:
      replicas: 1
      restartPolicy: OnFailure
      template:
        spec:
          containers:
            - name: pytorch
              image: kubeflow/pytorch-dist-mnist-test:1.0
              args: ["--backend", "gloo"]
              #ports:
              #- port: 23456
              #  name: pytorch-job-port
              # Comment out the below resources to use the CPU.
              resources:
                limits:
                  nvidia.com/gpu: 1
    Worker:
      replicas: 1
      restartPolicy: OnFailure
      template:
        spec:
          containers:
            - name: pytorch
              image: kubeflow/pytorch-dist-mnist-test:1.0
              args: ["--backend", "gloo"]
              #ports:
              #- port: 23456
              #  name: pytorch-job-port
              # Comment out the below resources to use the CPU.
              resources:
                limits:
                  nvidia.com/gpu: 1
```
Then Test with the same command:

```
shell
kubectl create -f ./v1beta1/pytorch_job_mnist_gloo.yaml
```

