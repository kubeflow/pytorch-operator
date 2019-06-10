FROM registry.access.redhat.com/ubi8/ubi:latest

COPY pytorch-operator.v1beta2 /pytorch-operator.v1beta2
COPY pytorch-operator.v1 /pytorch-operator.v1

ENTRYPOINT ["/pytorch-operator", "-alsologtostderr"]
