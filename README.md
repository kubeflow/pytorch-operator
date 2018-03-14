
# pytorch-operator
#### Experimental repo notice: This repository is experimental and currently only serves as a proof of concept for running distributed training with PyTorch on Kubernetes. Current POC is based on [TFJob operator](https://github.com/kubeflow/tf-operator)
Repository for supporting pytorch. This repo is experimental and is being used to start work related to this [proposal](https://github.com/kubeflow/community/pull/33).

## Prerequisites

- [Helm and Tiller](https://github.com/kubernetes/helm/blob/master/docs/install.md)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl)
## Using the PyTorch Operator

Run the following to deploy the operator to the namespace of your current context:
```
RBAC=true #set false if you do not have an RBAC cluster
helm install pytorch-operator-chart -n pytorch-operator --set rbac.install=${RBAC} --wait --replace
```
For this POC example we will use a configmap that contains our distributed training script.
```
kubectl create -f examples/multinode/configmap.yaml
```
Create a PyTorchJob resource to start training:
```
kubectl create -f examples/crd.yaml
```
You should now be able to see the job running based on the specified number of replicas.
```
kubectl get pods -a -l pytorch_job_name=example-job
```
Training should run for about 10 epochs and takes 5-10 minutes on a cpu cluster. Logs don't appear until job is completed. (TODO(jose5918) Find a better example for distributed training)

Tail the logs and once the training job is completed the logs will show up:
```
PODNAME=$(kubectl  get pods -a -l pytorch_job_name=example-job,task_index=0 -o name)
kubectl logs -f ${PODNAME}
```
Example output:
```
Downloading http://yann.lecun.com/exdb/mnist/train-images-idx3-ubyte.gz
Downloading http://yann.lecun.com/exdb/mnist/train-labels-idx1-ubyte.gz
Downloading http://yann.lecun.com/exdb/mnist/t10k-images-idx3-ubyte.gz
Downloading http://yann.lecun.com/exdb/mnist/t10k-labels-idx1-ubyte.gz
Processing...
Done!
Rank  0 , epoch  0 :  1.2753884393269066
Rank  0 , epoch  1 :  0.5752273188915842
Rank  0 , epoch  2 :  0.4370715184919616
Rank  0 , epoch  3 :  0.37090928852558136
Rank  0 , epoch  4 :  0.3224359404430715
Rank  0 , epoch  5 :  0.29541213348158385
Rank  0 , epoch  6 :  0.27593734307583967
Rank  0 , epoch  7 :  0.25898529327055536
Rank  0 , epoch  8 :  0.24815570648862864
Rank  0 , epoch  9 :  0.22647559368756534
```