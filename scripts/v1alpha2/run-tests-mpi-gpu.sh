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

# This shell script is used to build a cluster and create a namespace from our
# argo workflow


set -o errexit
set -o nounset
set -o pipefail

CLUSTER_NAME="${CLUSTER_NAME}"
ZONE="${GCP_ZONE}"
PROJECT="${GCP_PROJECT}"
NAMESPACE="${DEPLOY_NAMESPACE}"
REGISTRY="${GCP_REGISTRY}"
VERSION=$(git describe --tags --always --dirty)
GO_DIR=${GOPATH}/src/github.com/${REPO_OWNER}/${REPO_NAME}
APP_NAME=test-app
KUBEFLOW_VERSION=master
KF_ENV=pytorch

echo "Activating service-account"
gcloud auth activate-service-account --key-file=${GOOGLE_APPLICATION_CREDENTIALS}
echo "Configuring kubectl"
gcloud --project ${PROJECT} container clusters get-credentials ${CLUSTER_NAME} \
    --zone ${ZONE}

cd ${APP_NAME}
echo "Install PyTorch v1alpha2 operator"
#/usr/local/bin/ks generate pytorch-operator pytorch-operator --pytorchJobImage=${REGISTRY}/${REPO_NAME}:${VERSION}
/usr/local/bin/ks param set pytorch-operator pytorchJobVersion v1alpha2
/usr/local/bin/ks apply ${KF_ENV} -c pytorch-operator

TIMEOUT=30
until kubectl get pods -n ${NAMESPACE} | grep pytorch-operator | grep 1/1 || [[ $TIMEOUT -eq 1 ]]; do
  sleep 10
  TIMEOUT=$(( TIMEOUT - 1 ))
done

pushd ${GO_DIR}

echo "Running GPU test"
REGISTRY="docker.io"
GPU_TEST_IMAGE="akado2009/pytorch-mpi-mnist-gpu"
go run ./test/e2e/v1alpha2/defaults.go --namespace=${NAMESPACE} --image=${REGISTRY}/${GPU_TEST_IMAGE} --name=gputestjob-cleannone

popd
