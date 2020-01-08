# Kubeflow PyTorchJob SDK
Python SDK for PyTorch-Operator

## Requirements.

Python 2.7 and 3.5+

## Installation & Usage
### pip install

```sh
pip install kubeflow-pytorchjob
```

Then import the package:
```python
from kubeflow import pytorchjob 
```

### Setuptools

Install via [Setuptools](http://pypi.python.org/pypi/setuptools).

```sh
python setup.py install --user
```
(or `sudo python setup.py install` to install the package for all users)


## Getting Started

Please follow the [sample](examples/kubeflow-pytorchjob-sdk.ipynb) to create, update and delete PyTorchJob.

## Documentation for API Endpoints

Class | Method | Description
------------ | ------------- | -------------
[PyTorchJobClient](docs/PyTorchJobClient.md) | [create](docs/PyTorchJobClient.md#create) | Create PyTorchJob|
[PyTorchJobClient](docs/PyTorchJobClient.md) | [get](docs/PyTorchJobClient.md#get)    | Get the specified PyTorchJob or all PyTorchJob in the namespace |
[PyTorchJobClient](docs/PyTorchJobClient.md) | [patch](docs/PyTorchJobClient.md#patch)  | Patch the specified PyTorchJob|
[PyTorchJobClient](docs/PyTorchJobClient.md) | [delete](docs/PyTorchJobClient.md#delete) | Delete the specified PyTorchJob |
[PyTorchJobClient](docs/PyTorchJobClient.md)  | [wait_for_job](docs/PyTorchJobClient.md#wait_for_job) | Wait for the specified job to finish |
[PyTorchJobClient](docs/PyTorchJobClient.md)  | [wait_for_condition](docs/PyTorchJobClient.md#wait_for_condition) | Waits until any of the specified conditions occur |
[PyTorchJobClient](docs/PyTorchJobClient.md)  | [get_job_status](docs/PyTorchJobClient.md#get_job_status) | Get the PyTorchJob status|
[PyTorchJobClient](docs/PyTorchJobClient.md)  | [is_job_running](docs/PyTorchJobClient.md#is_job_running) | Check if the PyTorchJob running |
[PyTorchJobClient](docs/PyTorchJobClient.md)  | [is_job_succeeded](docs/PyTorchJobClient.md#is_job_succeeded) | Check if the PyTorchJob Succeeded |

## Documentation For Models

 - [V1JobCondition](docs/V1JobCondition.md)
 - [V1JobStatus](docs/V1JobStatus.md)
 - [V1PyTorchJob](docs/V1PyTorchJob.md)
 - [V1PyTorchJobList](docs/V1PyTorchJobList.md)
 - [V1PyTorchJobSpec](docs/V1PyTorchJobSpec.md)
 - [V1ReplicaSpec](docs/V1ReplicaSpec.md)
 - [V1ReplicaStatus](docs/V1ReplicaStatus.md)
