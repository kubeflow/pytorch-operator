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

package trainer

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sErrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	torchv1alpha1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1alpha1"
	"github.com/kubeflow/pytorch-operator/pkg/util/k8sutil"
	// TOOO(jlewi): Rename to apiErrors
	"github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/helper"
	"github.com/kubeflow/pytorch-operator/pkg/util"
)

const (
	SuccessfulCreateReason = "SuccessfulCreate"
	FailedCreateReason     = "FailedCreate"
)

// PyTorchReplicaSet is a set of PyTorch processes all acting as the same role (e.g. worker
type PyTorchReplicaSet struct {
	ClientSet kubernetes.Interface
	recorder  record.EventRecorder
	// Job is a pointer to the TrainingJob to which this replica belongs.
	Job  *TrainingJob
	Spec torchv1alpha1.PyTorchReplicaSpec
}

// PyTorchReplicas is an interface for managing a set of replicas.
type PyTorchReplicaSetInterface interface {
	Create() error
	Delete() error
	GetStatus() (torchv1alpha1.PyTorchReplicaStatus, error)
}

// PyTorchConfig is a struct representing the TensorFlow config. This struct is turned into an environment
// which is used by TensorFlow processes to configure themselves.
type PyTorchConfig struct {
	Cluster     ClusterSpec `json:"cluster"`
	Task        TaskSpec    `json:"task"`
	Environment string      `json:"environment"`
}

func NewPyTorchReplicaSet(clientSet kubernetes.Interface, recorder record.EventRecorder, tfReplicaSpec torchv1alpha1.PyTorchReplicaSpec, job *TrainingJob) (*PyTorchReplicaSet, error) {
	if tfReplicaSpec.PyTorchReplicaType == torchv1alpha1.MASTER && *tfReplicaSpec.Replicas != 1 {
		return nil, errors.New("The MASTER must have Replicas = 1")
	}

	if tfReplicaSpec.MasterPort == nil {
		return nil, errors.New("tfReplicaSpec.MasterPort can't be nil.")
	}

	// Make sure the replica type is valid.
	validReplicaTypes := []torchv1alpha1.PyTorchReplicaType{torchv1alpha1.MASTER, torchv1alpha1.WORKER}

	isValidReplicaType := false
	for _, t := range validReplicaTypes {
		if t == tfReplicaSpec.PyTorchReplicaType {
			isValidReplicaType = true
			break
		}
	}

	if !isValidReplicaType {
		return nil, fmt.Errorf("tfReplicaSpec.PyTorchReplicaType is %v but must be one of %v", tfReplicaSpec.PyTorchReplicaType, validReplicaTypes)
	}

	return &PyTorchReplicaSet{
		ClientSet: clientSet,
		recorder:  recorder,
		Job:       job,
		Spec:      tfReplicaSpec,
	}, nil
}

// Labels returns the labels for this replica set.
func (s *PyTorchReplicaSet) Labels() KubernetesLabels {
	return KubernetesLabels(map[string]string{
		"kubeflow.org": "",
		"job_type":     string(s.Spec.PyTorchReplicaType),
		// runtime_id is set by Job.setup, which is called after the PyTorchReplicaSet is created.
		// this is why labels aren't a member variable.
		"runtime_id":       s.Job.job.Spec.RuntimeId,
		"pytorch_job_name": s.Job.job.ObjectMeta.Name})
}

func (s *PyTorchReplicaSet) Create(config *torchv1alpha1.ControllerConfig, worldSize int32) error {
	// Create services
	err := s.SyncServices()
	if err != nil {
		return err
	}

	// Create pods
	return s.SyncPods(worldSize)
}

// CreateServiceWithIndex will create a new service with specify index
func (s *PyTorchReplicaSet) CreateServiceWithIndex(index int32) (*v1.Service, error) {
	taskLabels := s.Labels()
	taskLabels["task_index"] = fmt.Sprintf("%v", index)

	// Create the service.
	service := &v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   s.genName(index),
			Labels: taskLabels,
			OwnerReferences: []meta_v1.OwnerReference{
				helper.AsOwner(s.Job.job),
			},
		},
		Spec: v1.ServiceSpec{
			Selector: taskLabels,
			Ports: []v1.ServicePort{
				{
					Name: "tf-port",
					Port: *s.Spec.MasterPort,
				},
			},
		},
	}

	log.Infof("Creating service: %v", service.ObjectMeta.Name)
	return s.ClientSet.CoreV1().Services(s.Job.job.ObjectMeta.Namespace).Create(service)
}

// CreatePodWithIndex will create a new pod with specify index
func (s *PyTorchReplicaSet) CreatePodWithIndex(index int32, worldSize int32) (*v1.Pod, error) {
	taskLabels := s.Labels()
	taskLabels["task_index"] = fmt.Sprintf("%v", index)

	pod := &v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   s.genPodName(index),
			Labels: taskLabels,
			OwnerReferences: []meta_v1.OwnerReference{
				helper.AsOwner(s.Job.job),
			},
		},
		Spec: *s.Spec.Template.Spec.DeepCopy(),
	}

	pod.Spec.SchedulerName = s.Job.SchedulerName()

	// Configure the PyTorch distributed environment variables
	masterPort := strconv.Itoa(int(*s.Spec.MasterPort))
	masterAddr := fmt.Sprintf("%v-%v-%v-%v", fmt.Sprintf("%.40s", s.Job.job.ObjectMeta.Name), "master", s.Job.job.Spec.RuntimeId, 0)
	if index == 0 {
		masterAddr = "localhost"
	}
	rank := strconv.Itoa(int(index))
	tfConfig := PyTorchConfig{
		Cluster: s.Job.ClusterSpec(),
		Task: TaskSpec{
			Type:  strings.ToLower(string(s.Spec.PyTorchReplicaType)),
			Index: int(index),
		},
		// We need to set environment to cloud  otherwise it will default to local which isn't what we want.
		Environment: "cloud",
	}

	tfConfigJson, err := json.Marshal(tfConfig)
	if err != nil {
		log.Errorf("Job: %v serializing tfConfig: %v return error; %v", s.Job.job.ObjectMeta.Name, util.Pformat(tfConfig), err)
		return nil, err
	}

	// TODO(jose5918) Do not need TF_CONFIG but leaving for POC
	// Add TF_CONFIG environment variable.
	for i, _ := range pod.Spec.Containers {
		// We can't get c in the loop variable because that would be by value so our modifications
		// wouldn't have any effect.
		c := &pod.Spec.Containers[i]
		if c.Name != torchv1alpha1.DefaultPyTorchContainer {
			continue
		}
		if len(c.Env) == 0 {
			c.Env = make([]v1.EnvVar, 0)
		}
		c.Env = append(c.Env, v1.EnvVar{
			Name:  "TF_CONFIG",
			Value: string(tfConfigJson),
		})
		c.Env = append(c.Env, v1.EnvVar{
			Name:  "MASTER_PORT",
			Value: masterPort,
		})
		c.Env = append(c.Env, v1.EnvVar{
			Name:  "MASTER_ADDR",
			Value: masterAddr,
		})
		c.Env = append(c.Env, v1.EnvVar{
			Name:  "WORLD_SIZE",
			Value: strconv.Itoa(int(worldSize)),
		})
		c.Env = append(c.Env, v1.EnvVar{
			Name:  "RANK",
			Value: rank,
		})
	}

	log.Infof("Creating pod: %v", pod.ObjectMeta.Name)
	return s.ClientSet.CoreV1().Pods(s.Job.job.ObjectMeta.Namespace).Create(pod)
}

// Delete deletes the replicas
func (s *PyTorchReplicaSet) Delete() error {
	selector, err := s.Labels().ToSelector()
	if err != nil {
		return err
	}

	failures := false

	options := meta_v1.ListOptions{
		LabelSelector: selector,
	}

	log.V(1).Infof("Deleting Jobs namespace=%v selector=%v", s.Job.job.ObjectMeta.Namespace, selector)
	err = s.ClientSet.CoreV1().Pods(s.Job.job.ObjectMeta.Namespace).DeleteCollection(&meta_v1.DeleteOptions{}, options)

	if err != nil {
		log.Errorf("There was a problem deleting the jobs; %v", err)
		failures = true
	}

	// We need to delete the completed pods.
	log.Infof("Deleting Pods namespace=%v selector=%v", s.Job.job.ObjectMeta.Namespace, selector)
	err = s.ClientSet.CoreV1().Pods(s.Job.job.ObjectMeta.Namespace).DeleteCollection(&meta_v1.DeleteOptions{}, options)

	if err != nil {
		log.Errorf("There was a problem deleting the pods; %v", err)
		failures = true
	}

	// Services doesn't support DeleteCollection so we delete them individually.
	// TODO(jlewi): We should check if this has changed with K8s 1.8 or other releases.
	for index := int32(0); index < *s.Spec.Replicas; index++ {
		log.V(1).Infof("Deleting Service %v:%v", s.Job.job.ObjectMeta.Namespace, s.genName((index)))
		err = s.ClientSet.CoreV1().Services(s.Job.job.ObjectMeta.Namespace).Delete(s.genName(index), &meta_v1.DeleteOptions{})

		if err != nil {
			log.Errorf("Error deleting service %v; %v", s.genName(index), err)
			failures = true
		}
	}

	// If the ConfigMap for the default parameter server exists, we delete it
	log.Infof("Get ConfigMaps %v:%v", s.Job.job.ObjectMeta.Namespace, s.defaultPSConfigMapName())
	_, err = s.ClientSet.CoreV1().ConfigMaps(s.Job.job.ObjectMeta.Namespace).Get(s.defaultPSConfigMapName(), meta_v1.GetOptions{})
	if err != nil {
		if !k8sutil.IsKubernetesResourceNotFoundError(err) {
			log.Errorf("Error deleting ConfigMap %v; %v", s.defaultPSConfigMapName(), err)
			failures = true
		}
	} else {
		log.Infof("Delete ConfigMaps %v:%v", s.Job.job.ObjectMeta.Namespace, s.defaultPSConfigMapName())
		err = s.ClientSet.CoreV1().ConfigMaps(s.Job.job.ObjectMeta.Namespace).Delete(s.defaultPSConfigMapName(), &meta_v1.DeleteOptions{})
		if err != nil {
			log.Errorf("There was a problem deleting the ConfigMaps; %v", err)
			failures = true
		}
	}

	if failures {
		return errors.New("Some of the replicas resources could not be deleted")
	}
	return nil
}

// replicaStatusFromPodList returns a status from a list of pods for a job.
func replicaStatusFromPodList(l v1.PodList, name string) torchv1alpha1.ReplicaState {
	var latest *v1.Pod
	for _, i := range l.Items {
		if latest == nil {
			latest = &i
			continue
		}
		if latest.Status.StartTime.Before(i.Status.StartTime) {
			latest = &i
		}
	}

	if latest == nil {
		return torchv1alpha1.ReplicaStateRunning
	}

	var tfState v1.ContainerState

	for _, i := range latest.Status.ContainerStatuses {
		if i.Name != name {
			continue
		}

		// We need to decide whether to use the current state or the previous termination state.
		tfState = i.State

		// If the container previously terminated we will look at the termination to decide whether it is a retryable
		// or permanenent error.
		if i.LastTerminationState.Terminated != nil {
			tfState = i.LastTerminationState
		}
	}

	if tfState.Running != nil || tfState.Waiting != nil {
		return torchv1alpha1.ReplicaStateRunning
	}

	if tfState.Terminated != nil {
		if tfState.Terminated.ExitCode == 0 {
			return torchv1alpha1.ReplicaStateSucceeded
		}

		if isRetryableTerminationState(tfState.Terminated) {
			// Since its a retryable error just return RUNNING.
			// We can just let Kubernetes restart the container to retry.
			return torchv1alpha1.ReplicaStateRunning
		}

		return torchv1alpha1.ReplicaStateFailed
	}

	return torchv1alpha1.ReplicaStateUnknown
}

func (s *PyTorchReplicaSet) GetSingleReplicaStatus(index int32) torchv1alpha1.ReplicaState {
	p, err := s.ClientSet.CoreV1().Pods(s.Job.job.ObjectMeta.Namespace).Get(s.genName(index), meta_v1.GetOptions{})

	if err != nil {
		return torchv1alpha1.ReplicaStateUnknown
	}

	if v1.PodSucceeded == p.Status.Phase {
		return torchv1alpha1.ReplicaStateSucceeded
	}

	labels := s.Labels()
	labels["task_index"] = fmt.Sprintf("%v", index)
	selector, err := labels.ToSelector()
	if err != nil {
		log.Errorf("labels.ToSelector() error; %v", err)
		return torchv1alpha1.ReplicaStateFailed
	}

	// TODO(jlewi): Handle errors. We need to get the pod and looking at recent container exits.
	l, err := s.ClientSet.CoreV1().Pods(s.Job.job.ObjectMeta.Namespace).List(meta_v1.ListOptions{
		// TODO(jlewi): Why isn't the label selector working?
		LabelSelector: selector,
	})

	if err != nil {
		// TODO(jlewi): Are there errors that should be treated as retryable errors?
		return torchv1alpha1.ReplicaStateFailed
	}

	status := replicaStatusFromPodList(*l, torchv1alpha1.DefaultPyTorchContainer)
	return status
}

// Status returns the status of the replica set.
func (s *PyTorchReplicaSet) GetStatus() (torchv1alpha1.PyTorchReplicaStatus, error) {
	status := torchv1alpha1.PyTorchReplicaStatus{
		PyTorchReplicaType: s.Spec.PyTorchReplicaType,
		State:              torchv1alpha1.ReplicaStateUnknown,
		ReplicasStates:     make(map[torchv1alpha1.ReplicaState]int),
	}

	increment := func(state torchv1alpha1.ReplicaState) {
		v, ok := status.ReplicasStates[state]
		if ok {
			status.ReplicasStates[state] = v + 1
		} else {
			status.ReplicasStates[state] = 1
		}
	}

	for index := int32(0); index < *s.Spec.Replicas; index++ {
		increment(s.GetSingleReplicaStatus(index))
	}

	// Determine the overall status for the replica set based on the status of the individual
	// replicas.
	// If any of the replicas failed mark the set as failed.
	if _, ok := status.ReplicasStates[torchv1alpha1.ReplicaStateFailed]; ok {
		status.State = torchv1alpha1.ReplicaStateFailed
		return status, nil
	}

	// If any replicas are RUNNING mark it as RUNNING.
	if _, ok := status.ReplicasStates[torchv1alpha1.ReplicaStateRunning]; ok {
		status.State = torchv1alpha1.ReplicaStateRunning
		return status, nil
	}

	// If all of the replicas succeeded consider it success.
	if v, ok := status.ReplicasStates[torchv1alpha1.ReplicaStateSucceeded]; ok && int32(v) == *s.Spec.Replicas {
		status.State = torchv1alpha1.ReplicaStateSucceeded
		return status, nil
	}

	return status, nil
}

// SyncPods will try to check current pods for this PyTorchReplicaSet and try to make it as desired.
func (s *PyTorchReplicaSet) SyncPods(worldSize int32) error {
	for index := int32(0); index < *s.Spec.Replicas; index++ {

		// Label to get all pods of this PyTorchReplicaType + index
		labels := s.Labels()
		labels["task_index"] = fmt.Sprintf("%v", index)
		rank := index
		if labels["job_type"] == "WORKER" {
			rank = index + 1
		}
		labels["task_index"] = fmt.Sprintf("%v", rank)

		labelSelector, err := labels.ToSelector()
		if err != nil {
			return err
		}

		// Filter the unactive pods
		fieldSelector := "status.phase!=" + string(v1.PodFailed)
		//",deletionTimestamp!=nil"

		options := meta_v1.ListOptions{
			LabelSelector: labelSelector,
			FieldSelector: fieldSelector,
		}
		// List to get pods
		pl, err := s.ClientSet.CoreV1().Pods(s.Job.job.ObjectMeta.Namespace).List(options)

		if len(pl.Items) == 0 {
			log.Infof("Pod  not found, create new one.")
			// Create the pod
			createdPod, err := s.CreatePodWithIndex(rank, worldSize)

			// If the pod already exists do nothing.
			if err != nil {
				if k8s_errors.IsAlreadyExists(err) {
					log.Infof("Pod: %v already exists.", createdPod.ObjectMeta.Name)
					continue
				}
				s.recorder.Eventf(s.Job.job, v1.EventTypeWarning, FailedCreateReason, "Error creating: %v", err)
				return k8sErrors.NewAggregate([]error{fmt.Errorf("Creating pod %v returned error.", createdPod.ObjectMeta.Name), err})
			}

			s.recorder.Eventf(s.Job.job, v1.EventTypeNormal, SuccessfulCreateReason, "Created pod: %v", createdPod.Name)
			continue
		}

		if err != nil {
			// TODO: handing this error
			continue
		}
	}

	return nil
}

// SyncServices will try to check current services for this PyTorchReplicaSet and try to make it as desired.
func (s *PyTorchReplicaSet) SyncServices() error {
	for index := int32(0); index < *s.Spec.Replicas; index++ {
		_, err := s.ClientSet.CoreV1().Services(s.Job.job.ObjectMeta.Namespace).Get(s.genName(index), meta_v1.GetOptions{})
		if err != nil && k8s_errors.IsNotFound(err) {
			log.Infof("Service: %v not found, create new one.", s.genName(index))
			// Create the service
			createdService, err := s.CreateServiceWithIndex(index)

			// If the service already exists do nothing.
			if err != nil {
				if k8s_errors.IsAlreadyExists(err) {
					log.Infof("Service: %v already exists.", s.genName(index))
					continue
				}
				s.recorder.Eventf(s.Job.job, v1.EventTypeWarning, FailedCreateReason, "Error creating: %v", err)
				return k8sErrors.NewAggregate([]error{fmt.Errorf("Creating Service %v returned error.", createdService.ObjectMeta.Name), err})
			}

			s.recorder.Eventf(s.Job.job, v1.EventTypeNormal, SuccessfulCreateReason, "Created Service: %v", createdService.Name)
			continue
		}

		if err != nil {
			// TODO: handing this error
			continue
		}
	}

	return nil
}

func (s *PyTorchReplicaSet) genName(index int32) string {
	// Truncate tfjob name to 40 characters
	// The whole job name should be compliant with the DNS_LABEL spec, up to a max length of 63 characters
	// Thus genName(40 chars)-replicaType(6 chars)-runtimeId(4 chars)-index(4 chars), also leaving some spaces
	// See https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md
	return fmt.Sprintf("%v-%v-%v-%v", fmt.Sprintf("%.40s", s.Job.job.ObjectMeta.Name), strings.ToLower(string(s.Spec.PyTorchReplicaType)), s.Job.job.Spec.RuntimeId, index)
}

func (s *PyTorchReplicaSet) genPodName(index int32) string {
	// Generate a new pod name with random string
	return s.genName(index) + "-" + util.RandString(5)
}

func (s *PyTorchReplicaSet) defaultPSConfigMapName() string {
	return fmt.Sprintf("cm-ps-%v", s.Job.job.Spec.RuntimeId)
}
