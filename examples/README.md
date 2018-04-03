## PyTorch distributed examples

Here are examples of jobs we would aim to run with the operator.

1. A simple example of distributed MNIST with pytorch on kubernetes:
   ```
   kubectl apply -f mnist/multinode/
   ```

   The configmap used in the example was created using the distributed training script found in the mnist directory:
   ```
   kubectl create configmap dist-train --from-file=mnist/dist_train.py
   ```

2. An example of distributed CIFAR10 with pytorch on kubernetes:
   ```
   kubectl apply -f cifar10/
   ```

   For faster execution, pre-download the dataset to each of your cluster nodes and edit the
   cifar10/pytorchjob_cifar.yaml file to include the below "predownload" entries in the spec containers:
   ```
    spec:
      containers:
      - image: pytorch/pytorch:latest
        imagePullPolicy: IfNotPresent
        name: pytorch
        volumeMounts:
          - name: training-result
            mountPath: /tmp/result
          - name: entrypoint
            mountPath: /tmp/entrypoint
          - name: predownload                               <- Add this line
            mountpath: /data                                <- Add this line
        command: [/tmp/entrypoint/dist_train_cifar.py]
      restartPolicy: OnFailure
      volumes:
        - name: training-result
          emptyDir: {}
        - name: entrypoint
          configMap:
            name: dist-train-cifar
            defaultMode: 0755
        - name: predownload                                 <- Add this line
          hostPath:                                         <- Add this line
            path: [absolute_path_to_predownloaded_data]     <- Add this line and path
      restartPolicy: OnFailure
    ```

    The extra entries will need to be present for both MASTER and WORKER replica types.