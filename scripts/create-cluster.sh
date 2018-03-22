#!/bin/bash

# Copyright 2017 The Kubernetes Authors.
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

# This shell is used to auto generate some useful tools for k8s, such as lister,
# informer, deepcopy, defaulter and so on.

set -o errexit
set -o nounset
set -o pipefail

CLUSTER_NAME=$1
ZONE=$2
PROJECT=$3
NAMESPACE=$4


echo "Creating GPU cluster"
gcloud --project ${PROJECT} beta container clusters create ${CLUSTER_NAME} \
    --zone ${ZONE} \
    --accelerator type=nvidia-tesla-k80,count=1 \
    --cluster-version 1.9
echo "Configuring kubectl"
gcloud --project ${PROJECT} container clusters get-credentials ${CLUSTER_NAME} \
    --zone ${ZONE}
echo "Create Namespace"
kubectl create ns ${NAMESPACE}