# Tips

1. NVIDIA GPUs can now be via container level resource requirements using the resource name nvidia.com/gpu:
    ```
      resources:
        limits:
            nvidia.com/gpu: 2 # requesting 2 
    ```
    **Keep in mind!** The number of GPUs used by workers and master should be less of equal to the number of available GPUs on your cluster/system. If you should have less, then we recommend you to reduce the number of workers, or use master ony (in case you have 1 GPU).
    
2. If you have only 1 GPU you have to control number of process per node. Parameter ```nproc_per_node``` inside Dockerfile should be equal to the dist world size.

    ```
        torch.distributed.launch --nproc_per_node <number of process per node>
    ```
