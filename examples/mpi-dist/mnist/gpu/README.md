## Installation tips
1. You need to configure your node to utilize GPU. In order this can be done the following way: 
    * Install [nvidia-docker2](https://github.com/NVIDIA/nvidia-docker)
    * Connect to your MasterNode and set nvidia as the default run in `/etc/docker/daymon.json`:
        ```
        {
            "default-runtime": "nvidia",
            "runtimes": {
                "nvidia": {
                    "path": "/usr/bin/nvidia-container-runtime",
                    "runtimeArgs": []
                }
            }
        }
        ```
    * After that deploy nvidia-daemon to kubernetes: 
        ```bash
        kubectl create -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v1.11/nvidia-device-plugin.yml
        ```
        
2. NVIDIA GPUs can now be consumed via container level resource requirements using the resource name nvidia.com/gpu:
   
        
      resources:
        limits:
            nvidia.com/gpu: 2 # requesting 2 GPUs
        