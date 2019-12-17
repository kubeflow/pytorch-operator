# Copyright 2019 kubeflow.org.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os

# PyTorchJob K8S constants
PYTORCHJOB_GROUP = 'kubeflow.org'
PYTORCHJOB_KIND = 'PyTorchJob'
PYTORCHJOB_PLURAL = 'pytorchjobs'
PYTORCHJOB_VERSION = os.environ.get('PYTORCHJOB_VERSION', 'v1')

PYTORCH_LOGLEVEL = os.environ.get('PYTORCHJOB_LOGLEVEL', 'INFO').upper()

# How long to wait in seconds for requests to the ApiServer
APISERVER_TIMEOUT = 120
