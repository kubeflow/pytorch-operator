# PyTorchJobClient

> PyTorchJobClient(config_file=None, context=None, client_configuration=None, persist_config=True)

User can loads authentication and cluster information from kube-config file and stores them in kubernetes.client.configuration. Parameters are as following:

parameter |  Description
------------ | -------------
config_file | Name of the kube-config file. Defaults to `~/.kube/config`. Note that for the case that the SDK is running in cluster and you want to operate PyTorchJob in another remote cluster, user must set `config_file` to load kube-config file explicitly, e.g. `PyTorchJobClient(config_file="~/.kube/config")`. |
context |Set the active context. If is set to None, current_context from config file will be used.|
client_configuration | The kubernetes.client.Configuration to set configs to.|
persist_config | If True, config file will be updated when changed (e.g GCP token refresh).|


The APIs for PyTorchJobClient are as following:

Class | Method |  Description
------------ | ------------- | -------------
PyTorchJobClient| [create](#create) | Create PyTorchJob|
PyTorchJobClient | [get](#get)    | Get the specified PyTorchJob or all PyTorchJob in the namespace |
PyTorchJobClient | [patch](#patch)  | Patch the specified PyTorchJob|
PyTorchJobClient | [delete](#delete) | Delete the specified PyTorchJob |


## create
> create(pytorchjob, namespace=None)

Create the provided pytorchjob in the specified namespace

### Example

```python
from kubernetes.client import V1PodTemplateSpec
from kubernetes.client import V1ObjectMeta
from kubernetes.client import V1PodSpec
from kubernetes.client import V1Container
from kubernetes.client import V1ResourceRequirements

from kubeflow.pytorchjob import constants
from kubeflow.pytorchjob import utils
from kubeflow.pytorchjob import V1ReplicaSpec
from kubeflow.pytorchjob import V1PyTorchJob
from kubeflow.pytorchjob import V1PyTorchJobSpec
from kubeflow.pytorchjob import PyTorchJobClient

  container = V1Container(
    name="pytorch",
    image="gcr.io/kubeflow-ci/pytorch-dist-mnist-test:v1.0",
    args=["--backend","gloo"],
  )

  master = V1ReplicaSpec(
    replicas=1,
    restart_policy="OnFailure",
    template=V1PodTemplateSpec(
      spec=V1PodSpec(
        containers=[container]
      )
    )
  )

  worker = V1ReplicaSpec(
    replicas=1,
    restart_policy="OnFailure",
    template=V1PodTemplateSpec(
      spec=V1PodSpec(
        containers=[container]
        )
    )
  )

  pytorchjob = V1PyTorchJob(
    api_version="kubeflow.org/v1",
    kind="PyTorchJob",
    metadata=V1ObjectMeta(name="mnist", namespace='default'),
    spec=V1PyTorchJobSpec(
      clean_pod_policy="None",
      pytorch_replica_specs={"Master": master,
                             "Worker": worker}
    )
  )

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.create(pytorchjob)

```


### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
pytorchjob  | [V1PyTorchJob](V1PyTorchJob.md) | pytorchjob defination| Required |
namespace | str | Namespace for pytorchjob deploying to. If the `namespace` is not defined, will align with pytorchjob definition, or use current or default namespace if namespace is not specified in pytorchjob definition.  | Optional |

### Return type
object

## get
> get(name=None, namespace=None)

Get the created pytorchjob in the specified namespace

### Example

```python
from kubeflow.pytorchjob import pytorchjobClient

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.get('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | pytorchjob name. If the `name` is not specified, it will get all pytorchjobs in the namespace.| Optional. |
namespace | str | The pytorchjob's namespace. Defaults to current or default namespace.| Optional |


### Return type
object


## patch
> patch(name, pytorchjob, namespace=None)

Patch the created pytorchjob in the specified namespace.

Note that if you want to set the field from existing value to `None`, `patch` API may not work, you need to use [replace](#replace) API to remove the field value.

### Example

```python

pytorchjob = V1PyTorchJob(
    api_version="kubeflow.org/v1",
    ... #update something in PyTorchJob spec
)

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.patch('mnist', isvc)

```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
pytorchjob  | [V1PyTorchJob](V1PyTorchJob.md) | pytorchjob defination| Required |
namespace | str | The pytorchjob's namespace for patching. If the `namespace` is not defined, will align with pytorchjob definition, or use current or default namespace if namespace is not specified in pytorchjob definition. | Optional|

### Return type
object


## delete
> delete(name, namespace=None)

Delete the created pytorchjob in the specified namespace

### Example

```python
from kubeflow.pytorchjob import pytorchjobClient

pytorchjob_client = PyTorchJobClient()
pytorchjob_client.delete('mnist', namespace='kubeflow')
```

### Parameters
Name | Type |  Description | Notes
------------ | ------------- | ------------- | -------------
name  | str | pytorchjob name| |
namespace | str | The pytorchjob's namespace. Defaults to current or default namespace. | Optional|

### Return type
object
