
.PHONY: docker test

all: test

VERSION := $(shell git describe --tags --always --dirty)

IMAGE_NAME=pytorch-operator

docker:
	docker build \
		-t $(IMAGE_NAME):$(VERSION) \
		-t $(IMAGE_NAME):latest \
		.

prereq:
	go get -u \
		github.com/alecthomas/gometalinter \
		github.com/kubernetes/gengo/examples/deepcopy-gen
	gometalinter --install

build: prereq code-generation lint test
	go build -gcflags "-N -l" github.com/kubeflow/pytorch-operator/cmd/pytorch-operator

lint:
	gometalinter --config=linter_config.json ./pkg/...

test:
	go test -v --cover ./pkg/apis/...
	go test -v --cover ./pkg/trainer/...

code-generation:
	./hack/update-codegen.sh

push-image: build
	@ echo "activating service-account"
	gcloud auth activate-service-account --key-file=$(GOOGLE_APPLICATION_CREDENTIALS)
	@ echo "building container in gcloud"
	gcloud container builds submit . --tag=$(REGISTRY)/$(IMAGE_NAME):$(VERSION) --project=$(PROJECT)
