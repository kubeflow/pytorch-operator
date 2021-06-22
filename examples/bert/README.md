### Distributed BERT Examples

This folder contains an example where bert is trained. This example is also used for e2e testing.

The python script used to train bert with pytorch lightning.

**Note**: PyTorch job doesn’t work in a user namespace by default because of Istio [automatic sidecar injection](https://istio.io/v1.3/docs/setup/additional-setup/sidecar-injection/#automatic-sidecar-injection). In order to get it running, it needs annotation sidecar.istio.io/inject: "false" to disable it for either PyTorch pods or namespace.

**Build Image**

```shell
docker build -f Dockerfile -t <username>/pytorch-dist-bert:latest .
docker push <username>/pytorch-dist-bert:latest
```

**Create the BERT PyTorch job**

The below example uses the gloo backend.

```shell
kubectl create -f ./v1/pytorch_job_bert.yaml
```
