FROM pytorch/pytorch:0.4_cuda9_cudnn7

RUN pip install tensorboardX
ADD . /opt/pytorch_dist_mnist
ENTRYPOINT ["python", "/opt/pytorch_dist_mnist/dist_mnist_with_summary.py"]
