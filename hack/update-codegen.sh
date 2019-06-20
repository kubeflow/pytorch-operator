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

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${SCRIPT_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
${CODEGEN_PKG}/generate-groups.sh "defaulter,deepcopy,client,informer,lister" \
 github.com/kubeflow/pytorch-operator/pkg/client github.com/kubeflow/pytorch-operator/pkg/apis \
 pytorch:v1beta2,v1 \
 --go-header-file ${SCRIPT_ROOT}/hack/boilerplate/boilerplate.go.txt

echo "Generating defaulters for pytorch v1beta2"
 ${GOPATH}/bin/defaulter-gen --input-dirs github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1beta2 \
 -O zz_generated.defaults \
 --go-header-file ./hack/../hack/boilerplate/boilerplate.go.txt  \
 --output-package github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1beta2


echo "Generating defaulters for pytorch v1"
 ${GOPATH}/bin/defaulter-gen --input-dirs github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1 \
 -O zz_generated.defaults \
 --go-header-file ./hack/../hack/boilerplate/boilerplate.go.txt  \
 --output-package github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1

echo "Generating OpenAPI specification for pytorch/v1beta2"
${GOPATH}/bin/openapi-gen --input-dirs github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1beta2,k8s.io/api/core/v1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/util/intstr,k8s.io/apimachinery/pkg/version --output-package github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1beta2 --go-header-file hack/boilerplate/boilerplate.go.txt "$@"
cd - > /dev/null

echo "Generating OpenAPI specification for pytorch/v1"
${GOPATH}/bin/openapi-gen --input-dirs github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1,k8s.io/api/core/v1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/util/intstr,k8s.io/apimachinery/pkg/version --output-package github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1 --go-header-file hack/boilerplate/boilerplate.go.txt "$@"
cd - > /dev/null
