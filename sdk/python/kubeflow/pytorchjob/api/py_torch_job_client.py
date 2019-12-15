# Copyright 2019 The Kubeflow Authors.
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

from kubernetes import client, config

from kubeflow.pytorchjob.constants import constants
from kubeflow.pytorchjob.utils import utils


class PyTorchJobClient(object):

  def __init__(self, config_file=None, context=None, # pylint: disable=too-many-arguments
               client_configuration=None, persist_config=True):
    """
    PyTorchJob client constructor
    :param config_file: kubeconfig file, defaults to ~/.kube/config
    :param context: kubernetes context
    :param client_configuration: kubernetes configuration object
    :param persist_config:
    """
    if config_file or not utils.is_running_in_k8s():
      config.load_kube_config(
        config_file=config_file,
        context=context,
        client_configuration=client_configuration,
        persist_config=persist_config)
    else:
      config.load_incluster_config()

    self.api_instance = client.CustomObjectsApi()


  def create(self, pytorchjob, namespace=None):
    """
    Create the PyTorchJob
    :param pytorchjob: pytorchjob object
    :param namespace: defaults to current or default namespace
    :return: created pytorchjob
    """

    if namespace is None:
      namespace = utils.set_pytorchjob_namespace(pytorchjob)

    try:
      outputs = self.api_instance.create_namespaced_custom_object(
        constants.PYTORCHJOB_GROUP,
        constants.PYTORCHJOB_VERSION,
        namespace,
        constants.PYTORCHJOB_PLURAL,
        pytorchjob)
    except client.rest.ApiException as e:
      raise RuntimeError(
        "Exception when calling CustomObjectsApi->create_namespaced_custom_object:\
         %s\n" % e)

    return outputs

  def get(self, name=None, namespace=None):
    """
    Get the pytorchjob
    :param name: existing pytorchjob name
    :param namespace: defaults to current or default namespace
    :return: pytorchjob
    """
    if namespace is None:
      namespace = utils.get_default_target_namespace()

    if name:
      try:
        return self.api_instance.get_namespaced_custom_object(
            constants.PYTORCHJOB_GROUP,
            constants.PYTORCHJOB_VERSION,
            namespace,
            constants.PYTORCHJOB_PLURAL,
            name)
      except client.rest.ApiException as e:
        raise RuntimeError(
          "Exception when calling CustomObjectsApi->get_namespaced_custom_object:\
            %s\n" % e)
    else:
      try:
        return self.api_instance.list_namespaced_custom_object(
          constants.PYTORCHJOB_GROUP,
          constants.PYTORCHJOB_VERSION,
          namespace,
          constants.PYTORCHJOB_PLURAL)
      except client.rest.ApiException as e:
        raise RuntimeError(
          "Exception when calling CustomObjectsApi->list_namespaced_custom_object:\
          %s\n" % e)

  def patch(self, name, pytorchjob, namespace=None):
    """
    Patch existing pytorchjob
    :param name: existing pytorchjob name
    :param pytorchjob: patched pytorchjob
    :param namespace: defaults to current or default namespace
    :return: patched pytorchjob
    """
    if namespace is None:
      namespace = utils.set_pytorchjob_namespace(pytorchjob)

    try:
      outputs = self.api_instance.patch_namespaced_custom_object(
        constants.PYTORCHJOB_GROUP,
        constants.PYTORCHJOB_VERSION,
        namespace,
        constants.PYTORCHJOB_PLURAL,
        name,
        pytorchjob)
    except client.rest.ApiException as e:
      raise RuntimeError(
        "Exception when calling CustomObjectsApi->patch_namespaced_custom_object:\
         %s\n" % e)

    return outputs


  def delete(self, name, namespace=None):
    """
    Delete the pytorchjob
    :param name: pytorchjob name
    :param namespace: defaults to current or default namespace
    :return:
    """
    if namespace is None:
      namespace = utils.get_default_target_namespace()

    try:
      return self.api_instance.delete_namespaced_custom_object(
        constants.PYTORCHJOB_GROUP,
        constants.PYTORCHJOB_VERSION,
        namespace,
        constants.PYTORCHJOB_PLURAL,
        name,
        client.V1DeleteOptions())
    except client.rest.ApiException as e:
      raise RuntimeError(
        "Exception when calling CustomObjectsApi->delete_namespaced_custom_object:\
         %s\n" % e)
