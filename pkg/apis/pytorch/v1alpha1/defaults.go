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

package v1alpha1

import (
	"github.com/golang/protobuf/proto"
	"k8s.io/apimachinery/pkg/runtime"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_PyTorchJob sets any unspecified values to defaults
func SetDefaults_PyTorchJob(obj *PyTorchJob) {
	c := &obj.Spec

	if c.PyTorchImage == "" {
		c.PyTorchImage = DefaultPyTorchImage
	}

	// Check that each replica has a pytorch container.
	for _, r := range c.ReplicaSpecs {

		if r.MasterPort == nil {
			r.MasterPort = proto.Int32(MasterPort)
		}

		if string(r.PyTorchReplicaType) == "" {
			r.PyTorchReplicaType = MASTER
		}

		if r.Replicas == nil {
			r.Replicas = proto.Int32(Replicas)
		}
	}
	if c.TerminationPolicy == nil {
		c.TerminationPolicy = &TerminationPolicySpec{
			Master: &MasterSpec{
				ReplicaName: "MASTER",
				ReplicaRank: 0,
			},
		}
	}

}
