FROM golang:1.13 AS build-image

ADD . /go/src/github.com/kubeflow/pytorch-operator

WORKDIR /go/src/github.com/kubeflow/pytorch-operator

# Build pytorch operator v1 binary
RUN go build ./cmd/pytorch-operator.v1

FROM registry.access.redhat.com/ubi8/ubi:latest

COPY --from=build-image /go/src/github.com/kubeflow/pytorch-operator/pytorch-operator.v1 /pytorch-operator.v1

COPY third_party/library/license.txt /license.txt

RUN mkdir -p /vendor

ENTRYPOINT ["/pytorch-operator.v1", "-alsologtostderr"]
