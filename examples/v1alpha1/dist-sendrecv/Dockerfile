FROM pytorch/pytorch:v0.2

RUN mkdir -p /opt/mlkube
COPY dist_sendrecv.py /opt/mlkube/
ENTRYPOINT ["python", "/opt/mlkube/dist_sendrecv.py"]
