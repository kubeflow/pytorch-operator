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
	"errors"
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	pyv1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1"
	common "github.com/kubeflow/common/pkg/apis/common/v1"
	pylogger "github.com/kubeflow/tf-operator/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// pytorchJobCreatedReason is added in a job when it is created.
	pytorchJobCreatedReason = "PyTorchJobCreated"
	// pytorchJobSucceededReason is added in a job when it is succeeded.
	pytorchJobSucceededReason = "PyTorchJobSucceeded"
	// pytorchJobRunningReason is added in a job when it is running.
	pytorchJobRunningReason = "PyTorchJobRunning"
	// pytorchJobFailedReason is added in a job when it is failed.
	pytorchJobFailedReason = "PyTorchJobFailed"
	// pytorchJobRestarting is added in a job when it is restarting.
	pytorchJobRestartingReason = "PyTorchJobRestarting"
)

var (
	pytorchJobsSuccessCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pytorch_operator_jobs_successful_total",
		Help: "Counts number of PyTorch jobs successful",
	})
	pytorchJobsFailureCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pytorch_operator_jobs_failed_total",
		Help: "Counts number of PyTorch jobs failed",
	})
	pytorchJobsRestartCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pytorch_operator_jobs_restarted_total",
		Help: "Counts number of PyTorch jobs restarted",
	})
)

// updateStatus updates the status of the job.
func (pc *PyTorchController) updateStatusSingle(job *pyv1.PyTorchJob, rtype pyv1.PyTorchReplicaType, replicas int, restart bool) error {
	jobKey, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return err
	}

	// Expect to have `replicas - succeeded` pods alive.
	commonType := common.ReplicaType(rtype)
	expected := replicas - int(job.Status.ReplicaStatuses[commonType].Succeeded)
	running := int(job.Status.ReplicaStatuses[commonType].Active)
	failed := int(job.Status.ReplicaStatuses[commonType].Failed)

	pylogger.LoggerForJob(job).Infof("PyTorchJob=%s, ReplicaType=%s expected=%d, running=%d, failed=%d",
		job.Name, rtype, expected, running, failed)
	// Set StartTime.
	if job.Status.StartTime == nil {
		now := metav1.Now()
		job.Status.StartTime = &now
		// enqueue a sync to check if job past ActiveDeadlineSeconds
		if job.Spec.ActiveDeadlineSeconds != nil {
			pylogger.LoggerForJob(job).Infof("Job with ActiveDeadlineSeconds will sync after %d seconds", *job.Spec.ActiveDeadlineSeconds)
			pc.WorkQueue.AddAfter(jobKey, time.Duration(*job.Spec.ActiveDeadlineSeconds)*time.Second)
		}
	}

	if ContainMasterSpec(job) {
		if rtype == pyv1.PyTorchReplicaTypeMaster {
			if running > 0 {
				msg := fmt.Sprintf("PyTorchJob %s is running.", job.Name)
				err := updatePyTorchJobConditions(job, common.JobRunning, pytorchJobRunningReason, msg)
				if err != nil {
					pylogger.LoggerForJob(job).Infof("Append job condition error: %v", err)
					return err
				}
			}
			if expected == 0 {
				msg := fmt.Sprintf("PyTorchJob %s is successfully completed.", job.Name)
				pc.Recorder.Event(job, v1.EventTypeNormal, pytorchJobSucceededReason, msg)
				if job.Status.CompletionTime == nil {
					now := metav1.Now()
					job.Status.CompletionTime = &now
				}
				err := updatePyTorchJobConditions(job, common.JobSucceeded, pytorchJobSucceededReason, msg)
				if err != nil {
					pylogger.LoggerForJob(job).Infof("Append job condition error: %v", err)
					return err
				}
				pytorchJobsSuccessCount.Inc()
			}
		}
	} else {
		pylogger.LoggerForJob(job).Info("Invalid config: Job must contain master replica spec")
		return errors.New("invalid config: Job must contain master replica spec")
	}

	if failed > 0 {
		if restart {
			msg := fmt.Sprintf("PyTorchJob %s is restarting because %d %s replica(s) failed.", job.Name, failed, rtype)
			pc.Recorder.Event(job, v1.EventTypeWarning, pytorchJobRestartingReason, msg)
			err := updatePyTorchJobConditions(job, common.JobRestarting, pytorchJobRestartingReason, msg)
			if err != nil {
				pylogger.LoggerForJob(job).Infof("Append job condition error: %v", err)
				return err
			}
			pytorchJobsFailureCount.Inc()
			pytorchJobsRestartCount.Inc()
		} else {
			msg := fmt.Sprintf("PyTorchJob %s is failed because %d %s replica(s) failed.", job.Name, failed, rtype)
			pc.Recorder.Event(job, v1.EventTypeNormal, pytorchJobFailedReason, msg)
			if job.Status.CompletionTime == nil {
				now := metav1.Now()
				job.Status.CompletionTime = &now
			}
			err := updatePyTorchJobConditions(job, common.JobFailed, pytorchJobFailedReason, msg)
			if err != nil {
				pylogger.LoggerForJob(job).Infof("Append job condition error: %v", err)
				return err
			}
			pytorchJobsFailureCount.Inc()
		}
	}
	return nil
}

// updatePyTorchJobStatus updates the status of the given PyTorchJob.
func (pc *PyTorchController) updatePyTorchJobStatus(job *pyv1.PyTorchJob) error {
	_, err := pc.jobClientSet.KubeflowV1().PyTorchJobs(job.Namespace).UpdateStatus(job)
	return err
}

// updatePyTorchJobConditions updates the conditions of the given job.
func updatePyTorchJobConditions(job *pyv1.PyTorchJob, conditionType common.JobConditionType, reason, message string) error {
	condition := newCondition(conditionType, reason, message)
	setCondition(&job.Status, condition)
	return nil
}

// initializePyTorchReplicaStatuses initializes the PyTorchReplicaStatuses for replica.
func initializePyTorchReplicaStatuses(job *pyv1.PyTorchJob, rtype pyv1.PyTorchReplicaType) {
	commonType := common.ReplicaType(rtype)
	if job.Status.ReplicaStatuses == nil {
		job.Status.ReplicaStatuses = make(map[common.ReplicaType]*common.ReplicaStatus)
	}

	job.Status.ReplicaStatuses[commonType] = &common.ReplicaStatus{}
}

// updatePyTorchJobReplicaStatuses updates the PyTorchJobReplicaStatuses according to the pod.
func updatePyTorchJobReplicaStatuses(job *pyv1.PyTorchJob, rtype pyv1.PyTorchReplicaType, pod *v1.Pod) {
	commonType := common.ReplicaType(rtype)
	switch pod.Status.Phase {
	case v1.PodRunning:
		job.Status.ReplicaStatuses[commonType].Active++
	case v1.PodSucceeded:
		job.Status.ReplicaStatuses[commonType].Succeeded++
	case v1.PodFailed:
		job.Status.ReplicaStatuses[commonType].Failed++
	}
}

// newCondition creates a new job condition.
func newCondition(conditionType common.JobConditionType, reason, message string) common.JobCondition {
	return common.JobCondition{
		Type:               conditionType,
		Status:             v1.ConditionTrue,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// getCondition returns the condition with the provided type.
func getCondition(status common.JobStatus, condType common.JobConditionType) *common.JobCondition {
	for _, condition := range status.Conditions {
		if condition.Type == condType {
			return &condition
		}
	}
	return nil
}

func hasCondition(status common.JobStatus, condType common.JobConditionType) bool {
	for _, condition := range status.Conditions {
		if condition.Type == condType && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

func isSucceeded(status common.JobStatus) bool {
	return hasCondition(status, common.JobSucceeded)
}

func isFailed(status common.JobStatus) bool {
	return hasCondition(status, common.JobFailed)
}

// setCondition updates the job to include the provided condition.
// If the condition that we are about to add already exists
// and has the same status and reason then we are not going to update.
func setCondition(status *common.JobStatus, condition common.JobCondition) {
	// Do nothing if PyTorchJobStatus is completed
	if isFailed(*status) || isSucceeded(*status) {
		return
	}

	currentCond := getCondition(*status, condition.Type)

	// Do nothing if condition doesn't change
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	// Append the updated condition to the
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// filterOutCondition returns a new slice of job conditions without conditions with the provided type.
func filterOutCondition(conditions []common.JobCondition, condType common.JobConditionType) []common.JobCondition {
	var newConditions []common.JobCondition
	for _, c := range conditions {
		if condType == common.JobRestarting && c.Type == common.JobRunning {
			continue
		}
		if condType == common.JobRunning && c.Type == common.JobRestarting {
			continue
		}

		if c.Type == condType {
			continue
		}

		// Set the running condition status to be false when current condition failed or succeeded
		if (condType == common.JobFailed || condType == common.JobSucceeded) && c.Type == common.JobRunning {
			c.Status = v1.ConditionFalse
		}

		newConditions = append(newConditions, c)
	}
	return newConditions
}
