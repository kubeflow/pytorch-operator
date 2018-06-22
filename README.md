
# Kubernetes Custom Resource and Operator for PyTorch jobs

[![Build Status](https://travis-ci.org/kubeflow/pytorch-operator.svg?branch=master)](https://travis-ci.org/kubeflow/pytorch-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeflow/pytorch-operator)](https://goreportcard.com/report/github.com/kubeflow/pytorch-operator)

## Overview

This repository contains the specification and implementation of `PyTorchJob` custom resource definition. Using this custom resource, users can create and manage PyTorch jobs like other built-in resources in Kubernetes. See [CRD definition](https://github.com/kubeflow/kubeflow/blob/master/kubeflow/pytorch-job/pytorch-operator.libsonnet#L13)
  
## Prerequisites

- Kubernetes >= 1.8
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl)

## Installing PyTorch Operator

  Please refer to the installation instructions in the [Kubeflow user guide](https://www.kubeflow.org/docs/user_guide/#deploy-kubeflow). This installs `pytorchjob` CRD and `pytorch-operator` controller to manage the lifecycle of PyTorch jobs.

## Creating a PyTorch Job

You can create PyTorch Job by defining a PyTorchJob config file. See [distributed MNIST example](https://github.com/kubeflow/pytorch-operator/blob/master/examples/dist-mnist/pytorch_job_mnist.yaml) config file. You may change the config file based on your requirements.

```
cat examples/dist-mnist/pytorch_job_mnist.yaml
```
Deploy the PyTorchJob resource to start training:

```
kubectl create -f examples/dist-mnist/pytorch_job_mnist.yaml
```
You should now be able to see the created pods matching the specified number of replicas.

```
kubectl get pods -l pytorch_job_name=dist-mnist-for-e2e-test
```
Training should run for about 10 epochs and takes 5-10 minutes on a cpu cluster. Logs can be inspected to see its training progress. 

```
PODNAME=$(kubectl get pods -l pytorch_job_name=dist-mnist-for-e2e-test,task_index=0 -o name)
kubectl logs -f ${PODNAME}
```
## Monitoring a PyTorch Job

```
kubectl get -o yaml pytorchjobs dist-mnist-for-e2e-test
```
See the status section to monitor the job status. Here is sample output when the job is successfully completed.

```
apiVersion: v1
items:
- apiVersion: kubeflow.org/v1alpha1
  kind: PyTorchJob
  metadata:
    clusterName: ""
    creationTimestamp: 2018-06-22T08:16:14Z
    generation: 1
    name: dist-mnist-for-e2e-test
    namespace: default
    resourceVersion: "3276193"
    selfLink: /apis/kubeflow.org/v1alpha1/namespaces/default/pytorchjobs/dist-mnist-for-e2e-test
    uid: 87772d3b-75f4-11e8-bdd9-42010aa00072
  spec:
    RuntimeId: kmma
    pytorchImage: pytorch/pytorch:v0.2
    replicaSpecs:
    - masterPort: 23456
      replicaType: MASTER
      replicas: 1
      template:
        metadata:
          creationTimestamp: null
        spec:
          containers:
          - image: gcr.io/kubeflow-ci/pytorch-dist-mnist_test:1.0
            imagePullPolicy: IfNotPresent
            name: pytorch
            resources: {}
          restartPolicy: OnFailure
    - masterPort: 23456
      replicaType: WORKER
      replicas: 3
      template:
        metadata:
          creationTimestamp: null
        spec:
          containers:
          - image: gcr.io/kubeflow-ci/pytorch-dist-mnist_test:1.0
            imagePullPolicy: IfNotPresent
            name: pytorch
            resources: {}
          restartPolicy: OnFailure
    terminationPolicy:
      master:
        replicaName: MASTER
        replicaRank: 0
  status:
    phase: Done
    reason: ""
    replicaStatuses:
    - ReplicasStates:
        Succeeded: 1
      replica_type: MASTER
      state: Succeeded
    - ReplicasStates:
        Running: 1
        Succeeded: 2
      replica_type: WORKER
      state: Running
    state: Succeeded
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""

```
