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

// Package controller provides a Kubernetes controller for a PyTorchJob resource.
package pytorch

import (
	"testing"

	"k8s.io/api/core/v1"

	v1alpha2 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1alpha2"
	"github.com/kubeflow/pytorch-operator/pkg/util/testutil"
)

func TestFailed(t *testing.T) {
	job := testutil.NewPyTorchJobWithMaster(3)
	initializePyTorchReplicaStatuses(job, v1alpha2.PyTorchReplicaTypeWorker)
	pod := testutil.NewBasePod("pod", job, t)
	pod.Status.Phase = v1.PodFailed
	updatePyTorchJobReplicaStatuses(job, v1alpha2.PyTorchReplicaTypeWorker, pod)
	if job.Status.PyTorchReplicaStatuses[v1alpha2.PyTorchReplicaTypeWorker].Failed != 1 {
		t.Errorf("Failed to set the failed to 1")
	}
	err := updateStatusSingle(job, v1alpha2.PyTorchReplicaTypeWorker, 3, false)
	if err != nil {
		t.Errorf("Expected error %v to be nil", err)
	}
	found := false
	for _, condition := range job.Status.Conditions {
		if condition.Type == v1alpha2.PyTorchJobFailed {
			found = true
		}
	}
	if !found {
		t.Errorf("Failed condition is not found")
	}
}

func TestStatus(t *testing.T) {
	type testCase struct {
		description string
		job         *v1alpha2.PyTorchJob

		expectedFailedWorker    int32
		expectedSucceededWorker int32
		expectedActiveWorker    int32

		expectedFailedMaster    int32
		expectedSucceededMaster int32
		expectedActiveMaster    int32

		restart bool

		expectedType v1alpha2.PyTorchJobConditionType
	}

	testCases := []testCase{
		testCase{
			description:             "Master is succeeded",
			job:                     testutil.NewPyTorchJobWithMaster(1),
			expectedFailedWorker:    0,
			expectedSucceededWorker: 1,
			expectedActiveWorker:    0,
			expectedFailedMaster:    0,
			expectedSucceededMaster: 1,
			expectedActiveMaster:    0,
			restart:                 false,
			expectedType:            v1alpha2.PyTorchJobSucceeded,
		},
		testCase{
			description:             "Master is running",
			job:                     testutil.NewPyTorchJobWithMaster(1),
			expectedFailedWorker:    0,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    0,
			expectedFailedMaster:    0,
			expectedSucceededMaster: 0,
			expectedActiveMaster:    1,
			restart:                 false,
			expectedType:            v1alpha2.PyTorchJobRunning,
		},
		testCase{
			description:             "Master is failed",
			job:                     testutil.NewPyTorchJobWithMaster(1),
			expectedFailedWorker:    0,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    0,
			expectedFailedMaster:    1,
			expectedSucceededMaster: 0,
			expectedActiveMaster:    0,
			restart:                 false,
			expectedType:            v1alpha2.PyTorchJobFailed,
		},
		testCase{
			description:             "Master is running, workers are failed",
			job:                     testutil.NewPyTorchJobWithMaster(4),
			expectedFailedWorker:    4,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    0,
			expectedFailedMaster:    0,
			expectedSucceededMaster: 0,
			expectedActiveMaster:    1,
			restart:                 false,
			expectedType:            v1alpha2.PyTorchJobRunning,
		},
		testCase{
			description:             "Master is running, workers are succeeded",
			job:                     testutil.NewPyTorchJobWithMaster(4),
			expectedFailedWorker:    0,
			expectedSucceededWorker: 4,
			expectedActiveWorker:    0,
			expectedFailedMaster:    0,
			expectedSucceededMaster: 0,
			expectedActiveMaster:    1,
			restart:                 false,
			expectedType:            v1alpha2.PyTorchJobRunning,
		},
		testCase{
			description:             "Master is running, a worker is failed",
			job:                     testutil.NewPyTorchJobWithMaster(4),
			expectedFailedWorker:    1,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    3,
			expectedFailedMaster:    0,
			expectedSucceededMaster: 0,
			expectedActiveMaster:    1,
			expectedType:            v1alpha2.PyTorchJobFailed,
		},
		testCase{
			description:             "Master is failed, workers are succeeded",
			job:                     testutil.NewPyTorchJobWithMaster(4),
			expectedFailedWorker:    0,
			expectedSucceededWorker: 4,
			expectedActiveWorker:    0,
			expectedFailedMaster:    1,
			expectedSucceededMaster: 0,
			expectedActiveMaster:    0,
			restart:                 false,
			expectedType:            v1alpha2.PyTorchJobFailed,
		},
		testCase{
			description:             "Master is succeeded, workers are failed",
			job:                     testutil.NewPyTorchJobWithMaster(4),
			expectedFailedWorker:    4,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    0,
			expectedFailedMaster:    0,
			expectedSucceededMaster: 1,
			expectedActiveMaster:    0,
			restart:                 false,
			expectedType:            v1alpha2.PyTorchJobSucceeded,
		},
		testCase{
			description:             "Master is failed and restarting",
			job:                     testutil.NewPyTorchJobWithMaster(4),
			expectedFailedWorker:    4,
			expectedSucceededWorker: 0,
			expectedActiveWorker:    0,
			expectedFailedMaster:    1,
			expectedSucceededMaster: 0,
			expectedActiveMaster:    0,
			restart:                 true,
			expectedType:            v1alpha2.PyTorchJobRestarting,
		},
	}

	for i, c := range testCases {
		initializePyTorchReplicaStatuses(c.job, v1alpha2.PyTorchReplicaTypeWorker)
		initializePyTorchReplicaStatuses(c.job, v1alpha2.PyTorchReplicaTypeMaster)

		setStatusForTest(c.job, v1alpha2.PyTorchReplicaTypeMaster, c.expectedFailedMaster, c.expectedSucceededMaster, c.expectedActiveMaster, t)
		setStatusForTest(c.job, v1alpha2.PyTorchReplicaTypeWorker, c.expectedFailedWorker, c.expectedSucceededWorker, c.expectedActiveWorker, t)

		if _, ok := c.job.Spec.PyTorchReplicaSpecs[v1alpha2.PyTorchReplicaTypeMaster]; ok {
			err := updateStatusSingle(c.job, v1alpha2.PyTorchReplicaTypeMaster, 1, c.restart)
			if err != nil {
				t.Errorf("%s: Expected error %v to be nil", c.description, err)
			}
			if c.job.Spec.PyTorchReplicaSpecs[v1alpha2.PyTorchReplicaTypeWorker] != nil {
				replicas := c.job.Spec.PyTorchReplicaSpecs[v1alpha2.PyTorchReplicaTypeWorker].Replicas
				err := updateStatusSingle(c.job, v1alpha2.PyTorchReplicaTypeWorker, int(*replicas), c.restart)
				if err != nil {
					t.Errorf("%s: Expected error %v to be nil", c.description, err)
				}
			}

		}
		// Test filterOutCondition
		filterOutConditionTest(c.job.Status, t)

		found := false
		for _, condition := range c.job.Status.Conditions {
			if condition.Type == c.expectedType {
				found = true
			}
		}
		if !found {
			t.Errorf("Case[%d]%s: Condition %s is not found", i, c.description, c.expectedType)
		}
	}
}

func setStatusForTest(job *v1alpha2.PyTorchJob, typ v1alpha2.PyTorchReplicaType, failed, succeeded, active int32, t *testing.T) {
	pod := testutil.NewBasePod("pod", job, t)
	var i int32
	for i = 0; i < failed; i++ {
		pod.Status.Phase = v1.PodFailed
		updatePyTorchJobReplicaStatuses(job, typ, pod)
	}
	for i = 0; i < succeeded; i++ {
		pod.Status.Phase = v1.PodSucceeded
		updatePyTorchJobReplicaStatuses(job, typ, pod)
	}
	for i = 0; i < active; i++ {
		pod.Status.Phase = v1.PodRunning
		updatePyTorchJobReplicaStatuses(job, typ, pod)
	}
}

func filterOutConditionTest(status v1alpha2.PyTorchJobStatus, t *testing.T) {
	flag := isFailed(status) || isSucceeded(status)
	for _, condition := range status.Conditions {
		if flag && condition.Type == v1alpha2.PyTorchJobRunning && condition.Status == v1.ConditionTrue {
			t.Error("Error condition status when succeeded or failed")
		}
	}
}
