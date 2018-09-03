FROM debian:jessie

COPY pytorch-operator /pytorch-operator
COPY pytorch-operator.v2 /pytorch-operator.v2

ENTRYPOINT ["/pytorch-operator", "-alsologtostderr"]
