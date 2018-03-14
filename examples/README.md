## PyTorch distributed example

Here is an example of something we would aim to setup with the operator. It is just a simple example of distributed pytorch on kubernetes.
```
kubectl apply -f multinode/
```

The configmap used in the example was created using the distributed training script found in this directory:
```
kubectl create configmap dist-train --from-file=dist_train.py
```
