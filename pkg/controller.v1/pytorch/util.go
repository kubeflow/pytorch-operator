// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pytorch

import (
	"fmt"

	"bytes"
	"html/template"

	pyv1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1"
	"github.com/kubeflow/pytorch-operator/pkg/common/config"
	"github.com/kubernetes-sigs/yaml"
	v1 "k8s.io/api/core/v1"
)

var (
	errPortNotFound = fmt.Errorf("failed to found the port")
)

// GetPortFromPyTorchJob gets the port of pytorch container.
func GetPortFromPyTorchJob(job *pyv1.PyTorchJob, rtype pyv1.PyTorchReplicaType) (int32, error) {
	containers := job.Spec.PyTorchReplicaSpecs[rtype].Template.Spec.Containers
	for _, container := range containers {
		if container.Name == pyv1.DefaultContainerName {
			ports := container.Ports
			for _, port := range ports {
				if port.Name == pyv1.DefaultPortName {
					return port.ContainerPort, nil
				}
			}
		}
	}
	return -1, errPortNotFound
}

type InitContainerParam struct {
	MasterAddr         string
	InitContainerImage string
}

func ContainMasterSpec(job *pyv1.PyTorchJob) bool {
	if _, ok := job.Spec.PyTorchReplicaSpecs[pyv1.PyTorchReplicaTypeMaster]; ok {
		return true
	}
	return false
}

func GetInitContainer(containerTemplate string, param InitContainerParam) ([]v1.Container, error) {
	var buf bytes.Buffer
	tpl, err := template.New("container").Parse(containerTemplate)
	if err != nil {
		return nil, err
	}
	if err := tpl.Execute(&buf, param); err != nil {
		return nil, err
	}

	var result []v1.Container
	err = yaml.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func AddInitContainerForWorkerPod(podTemplate *v1.PodTemplateSpec, param InitContainerParam) error {
	containers, err := GetInitContainer(config.GetInitContainerTemplate(), param)
	if err != nil {
		return err
	}
	podTemplate.Spec.InitContainers = append(podTemplate.Spec.InitContainers, containers...)
	return nil
}
