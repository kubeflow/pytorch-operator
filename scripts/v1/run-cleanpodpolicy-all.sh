#!/bin/bash

# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This shell script is used to run a pytorch job with custom cleanpod policies


set -o errexit
set -o nounset
set -o pipefail

CLUSTER_NAME="${CLUSTER_NAME}"
REGION="${AWS_REGION:-us-west-2}"
NAMESPACE="${DEPLOY_NAMESPACE}"
REGISTRY="${GCP_REGISTRY}"
GO_DIR=${GOPATH}/src/github.com/kubeflow/${REPO_NAME}

echo "Configuring kubeconfig.."
aws eks update-kubeconfig --region=${REGION} --name=${CLUSTER_NAME}

cd ${GO_DIR}

echo "Running smoke test"
SENDRECV_TEST_IMAGE_TAG="pytorch-dist-sendrecv-test:v1.0"
go run ./test/e2e/v1/cleanpolicy/cleanpolicy_all.go --namespace=${NAMESPACE} --image=${REGISTRY}/${SENDRECV_TEST_IMAGE_TAG} --name=sendrecvjob-cleanall

# TODO(Jeffwan@): Enable mnist test once mnist server is back
#echo "Running mnist test"
#MNIST_TEST_IMAGE_TAG="pytorch-dist-mnist-test:v1.0"
#go run ./test/e2e/v1/cleanpolicy/cleanpolicy_all.go --namespace=${NAMESPACE} --image=${REGISTRY}/${MNIST_TEST_IMAGE_TAG} --name=mnistjob-cleanall
