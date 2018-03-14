FROM debian:jessie

COPY cmd/pytorch-operator/pytorch-operator /pytorch-operator

ENTRYPOINT ["/pytorch-operator", "-alsologtostderr"]