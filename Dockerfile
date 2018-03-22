FROM debian:jessie

COPY pytorch-operator /pytorch-operator

ENTRYPOINT ["/pytorch-operator", "-alsologtostderr"]
