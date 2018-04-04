## PyTorch distributed examples

Here are examples of jobs that use the operator.

1. An example of distributed CIFAR10 with pytorch on kubernetes:
   ```
   kubectl apply -f cifar10/
   ```

   For faster execution, pre-download the dataset to each of your cluster nodes and edit the
   cifar10/pytorchjob_cifar.yaml file to include the below "predownload" entries in the spec containers.
   The extra entries will need to be present for both MASTER and WORKER replica types.
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

2. A simple example of distributed MNIST with pytorch on kubernetes:
   ```
   kubectl apply -f mnist/
   ```
