FROM pytorch/pytorch:1.8.1-cuda10.2-cudnn7-runtime

RUN apt-get update && apt-get install -y git

ENV PATH=/root/.local/bin:$PATH

RUN pip install --user transformers torchtext pandas sklearn

RUN pip install --user git+https://github.com/PyTorchLightning/pytorch-lightning@master

RUN mkdir -p /workspace/bert/src/

ADD bert.py /workspace/bert/src/bert.py

ENTRYPOINT ["python", "/workspace/bert/src/bert.py"]
