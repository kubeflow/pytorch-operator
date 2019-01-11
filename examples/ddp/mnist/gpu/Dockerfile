FROM pytorch/pytorch:0.4_cuda9_cudnn7

ADD . /opt/pytorch_dist_mnist
ENTRYPOINT ["python", "-u", "-m", "torch.distributed.launch", "--nproc_per_node", "2", "/opt/pytorch_dist_mnist/mnist_ddp_gpu.py"]
