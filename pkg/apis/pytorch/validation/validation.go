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

package validation

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"

	torchv1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1alpha1"
	torchv2 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1alpha2"
	"github.com/kubeflow/pytorch-operator/pkg/util"
)

// ValidatePyTorchJobSpec checks that the PyTorchJobSpec is valid.
func ValidatePyTorchJobSpec(c *torchv1.PyTorchJobSpec) error {
	if c.TerminationPolicy == nil || c.TerminationPolicy.Master == nil {
		return fmt.Errorf("invalid termination policy: %v", c.TerminationPolicy)
	}

	masterExists := false

	// Check that each replica has a TensorFlow container and a master.
	for _, r := range c.ReplicaSpecs {
		found := false
		if r.Template == nil {
			return fmt.Errorf("Replica is missing Template; %v", util.Pformat(r))
		}

		if r.PyTorchReplicaType == torchv1.PyTorchReplicaType(c.TerminationPolicy.Master.ReplicaName) {
			masterExists = true
		}

		if r.MasterPort == nil {
			return errors.New("PyTorchReplicaSpec.MasterPort can't be nil.")
		}

		// Make sure the replica type is valid.
		validReplicaTypes := []torchv1.PyTorchReplicaType{torchv1.MASTER, torchv1.WORKER}

		isValidReplicaType := false
		for _, t := range validReplicaTypes {
			if t == r.PyTorchReplicaType {
				isValidReplicaType = true
				break
			}
		}

		if !isValidReplicaType {
			return fmt.Errorf("tfReplicaSpec.PyTorchReplicaType is %v but must be one of %v", r.PyTorchReplicaType, validReplicaTypes)
		}

		for _, c := range r.Template.Spec.Containers {
			if c.Name == torchv1.DefaultPyTorchContainer {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Replica type %v is missing a container named %s", r.PyTorchReplicaType, torchv1.DefaultPyTorchContainer)
		}
	}

	if !masterExists {
		return fmt.Errorf("Missing ReplicaSpec for master: %v", c.TerminationPolicy.Master.ReplicaName)
	}

	return nil
}

func ValidateAlphaTwoPyTorchJobSpec(c *torchv2.PyTorchJobSpec) error {
	if c.PyTorchReplicaSpecs == nil {
		return fmt.Errorf("PyTorchJobSpec is not valid")
	}
	masterExists := false
	for rType, value := range c.PyTorchReplicaSpecs {
		if value == nil || len(value.Template.Spec.Containers) == 0 {
			return fmt.Errorf("PyTorchJobSpec is not valid")
		}
		// Make sure the replica type is valid.
		validReplicaTypes := []torchv2.PyTorchReplicaType{torchv2.PyTorchReplicaTypeMaster, torchv2.PyTorchReplicaTypeWorker}

		isValidReplicaType := false
		for _, t := range validReplicaTypes {
			if t == rType {
				isValidReplicaType = true
				break
			}
		}

		if !isValidReplicaType {
			return fmt.Errorf("PyTorchReplicaType is %v but must be one of %v", rType, validReplicaTypes)
		}

		//Make sure the image is defined in the container
		defaultContainerPresent := false
		for _, container := range value.Template.Spec.Containers {
			if container.Image == "" {
				log.Warn("Image is undefined in the container")
				return fmt.Errorf("PyTorchJobSpec is not valid")
			}
			if container.Name == torchv2.DefaultContainerName {
				defaultContainerPresent = true
			}
		}
		//Make sure there has at least one container named "pytorch"
		if !defaultContainerPresent {
			log.Warnf("There is no container named pytorch in %v", rType)
			return fmt.Errorf("PyTorchJobSpec is not valid")
		}
		if rType == torchv2.PyTorchReplicaTypeMaster {
			masterExists = true
			if value.Replicas != nil && int(*value.Replicas) != 1 {
				log.Warnf("There must be only 1 master replica")
				return fmt.Errorf("PyTorchJobSpec is not valid")
			}
		}

	}

	if !masterExists {
		log.Warnf("Master ReplicaSpec must be present")
		return fmt.Errorf("PyTorchJobSpec is not valid")
	}
	return nil
}
