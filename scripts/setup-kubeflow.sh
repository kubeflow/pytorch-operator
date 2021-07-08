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
# TBD (@jinchihe) Since the ksonnet registry has been removed kubeflow master.
# Change to 0.7 version to work around, we need to enhance this later.
# See more https://github.com/kubeflow/pytorch-operator/issues/229
#KUBEFLOW_VERSION=master
KUBEFLOW_VERSION=v0.7.0
KF_ENV=pytorch

echo "Activating service-account"
gcloud auth activate-service-account --key-file=${GOOGLE_APPLICATION_CREDENTIALS}
echo "Configuring kubectl"
gcloud --project ${PROJECT} container clusters get-credentials ${CLUSTER_NAME} \
    --zone ${ZONE}

ACCOUNT=`gcloud config get-value account --quiet`
echo "Setting account ${ACCOUNT}"
kubectl create clusterrolebinding default-admin --clusterrole=cluster-admin --user=${ACCOUNT}

echo "Install ksonnet app in namespace ${NAMESPACE}"
/usr/local/bin/ks-13 init ${APP_NAME}
cd ${APP_NAME}
/usr/local/bin/ks-13 env add ${KF_ENV}
/usr/local/bin/ks-13 env set ${KF_ENV} --namespace ${NAMESPACE}
/usr/local/bin/ks-13 registry add kubeflow github.com/kubeflow/kubeflow/tree/${KUBEFLOW_VERSION}/kubeflow

echo "Install PyTorch ksonnet package"
/usr/local/bin/ks-13 pkg install kubeflow/pytorch-job@${KUBEFLOW_VERSION}

echo "Install PyTorch operator"
/usr/local/bin/ks-13 generate pytorch-operator pytorch-operator --pytorchJobImage=${REGISTRY}/${REPO_NAME}:${VERSION}
/usr/local/bin/ks-13 apply ${KF_ENV} -c pytorch-operator

TIMEOUT=30
until kubectl get pods -n ${NAMESPACE} | grep pytorch-operator | grep 1/1 || [[ $TIMEOUT -eq 1 ]]; do
  sleep 10
  TIMEOUT=$(( TIMEOUT - 1 ))
done
