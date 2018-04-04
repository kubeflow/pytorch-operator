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

echo "Activating service-account"
gcloud auth activate-service-account --key-file=${GOOGLE_APPLICATION_CREDENTIALS}
echo "Configuring kubectl"
gcloud --project ${PROJECT} container clusters get-credentials ${CLUSTER_NAME} \
    --zone ${ZONE}

echo "Deploying tiller"
kubectl create serviceaccount tiller -n kube-system
kubectl create clusterrolebinding tiller-cluster-admin-binding --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
helm init --service-account tiller
echo "Waiting for tiller"
TIMEOUT=30
until kubectl get pods -n kube-system | grep tiller-deploy | grep 1/1 || [[ $TIMEOUT -eq 1 ]]; do
  sleep 10
  TIMEOUT=$(( TIMEOUT - 1 ))
done

echo "Install the operator"
helm install pytorch-operator-chart -n pytorch-operator \
    --namespace ${NAMESPACE} \
    --set rbac.install=true \
    --set image=${REGISTRY}/${REPO_NAME}:${VERSION} \
    --wait --replace

echo "Run go tests"
cd ${GO_DIR}
kubectl apply -f examples/mnist/configmap.yaml -n ${NAMESPACE}
go run ./test/e2e/main.go --namespace=${NAMESPACE}
