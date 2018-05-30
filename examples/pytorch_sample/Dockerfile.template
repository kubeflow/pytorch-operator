FROM pytorch/pytorch:latest
RUN mkdir -p /opt/mlkube
COPY pytorch_smoke.py /opt/mlkube/
ENTRYPOINT ["python", "/opt/mlkube/pytorch_smoke.py"]
