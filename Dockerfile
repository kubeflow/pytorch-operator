FROM golang:1.12 AS build-image

ADD . /go/src/github.com/kubeflow/pytorch-operator
COPY ./vendor/github.com /usr/local/go/src/

WORKDIR /go/src/github.com/kubeflow/pytorch-operator

# Build pytorch operator v1beta2 binary
RUN go build github.com/kubeflow/pytorch-operator/cmd/pytorch-operator.v1beta2
# Build pytorch operator v1 binary
RUN go build github.com/kubeflow/pytorch-operator/cmd/pytorch-operator.v1

FROM registry.access.redhat.com/ubi8/ubi:latest

COPY --from=build-image /go/src/github.com/kubeflow/pytorch-operator/pytorch-operator.v1beta2 /pytorch-operator.v1beta2
COPY --from=build-image /go/src/github.com/kubeflow/pytorch-operator/pytorch-operator.v1 /pytorch-operator.v1

ENTRYPOINT ["/pytorch-operator", "-alsologtostderr"]
