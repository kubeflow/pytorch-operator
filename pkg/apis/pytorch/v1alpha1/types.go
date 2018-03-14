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
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CRDKind       = "pytorchjob"
	CRDKindPlural = "pytorchjobs"
	CRDGroup      = "kubeflow.org"
	CRDVersion    = "v1alpha1"
	// Value of the APP label that gets applied to a lot of entities.
	AppLabel = "pytorch-job"
	// Defaults for the Spec
	MasterPort = 23456
	Replicas   = 1
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=pytorchjob

// PyTorchJob describes pytorchjob info
type PyTorchJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PyTorchJobSpec   `json:"spec"`
	Status            PyTorchJobStatus `json:"status"`
}

type PyTorchJobSpec struct {
	// TODO(jlewi): Can we we get rid of this and use some value from Kubernetes or a random ide.
	RuntimeId string

	// ReplicaSpecs specifies the PyTorch replicas to run.
	ReplicaSpecs []*PyTorchReplicaSpec `json:"replicaSpecs"`

	// PyTorchImage defines the tensorflow docker image that should be used for default parameter server
	PyTorchImage string `json:"pytorchImage,omitempty"`

	// TerminationPolicy specifies the condition that the pytorchjob should be considered finished.
	TerminationPolicy *TerminationPolicySpec `json:"terminationPolicy,omitempty"`

	// SchedulerName specifies the name of scheduler which should handle the PyTorchJob
	SchedulerName string `json:"schedulerName,omitempty"`
}

type TerminationPolicySpec struct {
	// Master policy waits for a particular process (which is the master) to exit.
	Master *MasterSpec `json:"master,omitempty"`
}

type MasterSpec struct {
	ReplicaName string `json:"replicaName"`
	ReplicaRank int    `json:"replicaRank"`
}

// PyTorchReplicaType determines how a set of PyTorch processes are handled.
type PyTorchReplicaType string

const (
	MASTER PyTorchReplicaType = "MASTER"
	WORKER PyTorchReplicaType = "WORKER"
)

const (
	DefaultPyTorchContainer string = "pytorch"
	DefaultPyTorchImage     string = "pytorch/pytorch:v0.2"
)

// TODO(jlewi): We probably want to add a name field. This would allow us to have more than 1 type of each worker.
// This might be useful if you wanted to have a separate set of workers to do eval.
type PyTorchReplicaSpec struct {
	// Replicas is the number of desired replicas.
	// This is a pointer to distinguish between explicit zero and unspecified.
	// Defaults to 1.
	// More info: http://kubernetes.io/docs/user-guide/replication-controller#what-is-a-replication-controller
	// +optional
	Replicas *int32              `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`
	Template *v1.PodTemplateSpec `json:"template,omitempty" protobuf:"bytes,3,opt,name=template"`
	// MasterPort is the port to use for PyTorch services.
	MasterPort         *int32 `json:"masterPort,omitempty" protobuf:"varint,1,opt,name=masterPort"`
	PyTorchReplicaType `json:"replicaType"`
}

type PyTorchJobPhase string

const (
	PyTorchJobPhaseNone     PyTorchJobPhase = ""
	PyTorchJobPhaseCreating PyTorchJobPhase = "Creating"
	PyTorchJobPhaseRunning  PyTorchJobPhase = "Running"
	PyTorchJobPhaseCleanUp  PyTorchJobPhase = "CleanUp"
	PyTorchJobPhaseFailed   PyTorchJobPhase = "Failed"
	PyTorchJobPhaseDone     PyTorchJobPhase = "Done"
)

type State string

const (
	StateUnknown   State = "Unknown"
	StateRunning   State = "Running"
	StateSucceeded State = "Succeeded"
	StateFailed    State = "Failed"
)

type PyTorchJobStatus struct {
	// Phase is the PyTorchJob running phase
	Phase  PyTorchJobPhase `json:"phase"`
	Reason string          `json:"reason"`

	// State indicates the state of the job.
	State State `json:"state"`

	// ReplicaStatuses specifies the status of each PyTorch replica.
	ReplicaStatuses []*PyTorchReplicaStatus `json:"replicaStatuses"`
}

type ReplicaState string

const (
	ReplicaStateUnknown   ReplicaState = "Unknown"
	ReplicaStateRunning   ReplicaState = "Running"
	ReplicaStateFailed    ReplicaState = "Failed"
	ReplicaStateSucceeded ReplicaState = "Succeeded"
)

type PyTorchReplicaStatus struct {
	PyTorchReplicaType `json:"replica_type"`

	// State is the overall state of the replica
	State ReplicaState `json:"state"`

	// ReplicasStates provides the number of replicas in each status.
	ReplicasStates map[ReplicaState]int
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resource:path=pytorchjobs

// PyTorchJobList is a list of PyTorchJobs clusters.
type PyTorchJobList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`
	// Items is a list of PyTorchJobs
	Items []PyTorchJob `json:"items"`
}

type ControllerConfig struct {
	// Accelerators is a map from the name of the accelerator to the config for that accelerator.
	// This should match the value specified as a container limit.
	// e.g. alpha.kubernetes.io/nvidia-gpu
	Accelerators map[string]AcceleratorConfig

	// Path to the file containing the grpc server source
	GrpcServerFilePath string
}

// AcceleratorVolume represents a host path that must be mounted into
// each container that needs to use GPUs.
type AcceleratorVolume struct {
	Name      string
	HostPath  string
	MountPath string
}

type AcceleratorConfig struct {
	Volumes []AcceleratorVolume
	EnvVars []EnvironmentVariableConfig
}

type EnvironmentVariableConfig struct {
	Name  string
	Value string
}
