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
	"testing"

	torchv1 "github.com/jose5918/pytorch-operator/pkg/apis/pytorch/v1alpha1"

	"github.com/gogo/protobuf/proto"
	"k8s.io/api/core/v1"
)

func TestValidate(t *testing.T) {
	type testCase struct {
		in             *torchv1.PyTorchJobSpec
		expectingError bool
	}

	testCases := []testCase{
		{
			in: &torchv1.PyTorchJobSpec{
				ReplicaSpecs: []*torchv1.PyTorchReplicaSpec{
					{
						Template: &v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name: "pytorch",
									},
								},
							},
						},
						PyTorchReplicaType: torchv1.MASTER,
						Replicas:           proto.Int32(1),
					},
				},
				PyTorchImage: "pytorch/pytorch:v0.2",
			},
			expectingError: false,
		},
		{
			in: &torchv1.PyTorchJobSpec{
				ReplicaSpecs: []*torchv1.PyTorchReplicaSpec{
					{
						Template: &v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name: "pytorch",
									},
								},
							},
						},
						PyTorchReplicaType: torchv1.WORKER,
						Replicas:           proto.Int32(1),
					},
				},
				PyTorchImage: "pytorch/pytorch:v0.2",
			},
			expectingError: true,
		},
		{
			in: &torchv1.PyTorchJobSpec{
				ReplicaSpecs: []*torchv1.PyTorchReplicaSpec{
					{
						Template: &v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name: "pytorch",
									},
								},
							},
						},
						PyTorchReplicaType: torchv1.WORKER,
						Replicas:           proto.Int32(1),
					},
				},
				PyTorchImage: "pytorch/pytorch:v0.2",
				TerminationPolicy: &torchv1.TerminationPolicySpec{
					Master: &torchv1.MasterSpec{
						ReplicaName: "WORKER",
						ReplicaRank: 0,
					},
				},
			},
			expectingError: false,
		},
	}

	for _, c := range testCases {
		job := &torchv1.PyTorchJob{
			Spec: *c.in,
		}
		torchv1.SetObjectDefaults_PyTorchJob(job)
		if err := ValidatePyTorchJobSpec(&job.Spec); (err != nil) != c.expectingError {
			t.Errorf("unexpected validation result: %v", err)
		}
	}
}
