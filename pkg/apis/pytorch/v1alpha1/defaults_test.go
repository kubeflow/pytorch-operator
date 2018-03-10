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
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/jose5918/pytorch-operator/pkg/util"
	"k8s.io/api/core/v1"
)

func TestSetDefaults_PyTorchJob(t *testing.T) {
	type testCase struct {
		in       *PyTorchJob
		expected *PyTorchJob
	}

	testCases := []testCase{
		{
			in: &PyTorchJob{
				Spec: PyTorchJobSpec{
					ReplicaSpecs: []*PyTorchReplicaSpec{
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
						},
					},
					PyTorchImage: "pytorch/pytorch:v0.2",
				},
			},
			expected: &PyTorchJob{
				Spec: PyTorchJobSpec{
					ReplicaSpecs: []*PyTorchReplicaSpec{
						{
							Replicas:   proto.Int32(1),
							MasterPort: proto.Int32(23456),
							Template: &v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name: "pytorch",
										},
									},
								},
							},
							PyTorchReplicaType: MASTER,
						},
					},
					PyTorchImage: "pytorch/pytorch:v0.2",
					TerminationPolicy: &TerminationPolicySpec{
						Master: &MasterSpec{
							ReplicaName: "MASTER",
							ReplicaRank: 0,
						},
					},
				},
			},
		},
		{
			in: &PyTorchJob{
				Spec: PyTorchJobSpec{
					ReplicaSpecs: []*PyTorchReplicaSpec{
						{
							PyTorchReplicaType: WORKER,
						},
					},
					PyTorchImage: "pytorch/pytorch:v0.2",
				},
			},
			expected: &PyTorchJob{
				Spec: PyTorchJobSpec{
					ReplicaSpecs: []*PyTorchReplicaSpec{
						{
							Replicas:           proto.Int32(1),
							MasterPort:         proto.Int32(23456),
							PyTorchReplicaType: WORKER,
						},
					},
					PyTorchImage: "pytorch/pytorch:v0.2",
					TerminationPolicy: &TerminationPolicySpec{
						Master: &MasterSpec{
							ReplicaName: "MASTER",
							ReplicaRank: 0,
						},
					},
				},
			},
		},
	}

	for _, c := range testCases {
		SetDefaults_PyTorchJob(c.in)
		if !reflect.DeepEqual(c.in, c.expected) {
			t.Errorf("Want\n%v; Got\n %v", util.Pformat(c.expected), util.Pformat(c.in))
		}
	}
}
