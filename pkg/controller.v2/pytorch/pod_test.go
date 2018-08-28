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
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/controller"

	"github.com/kubeflow/pytorch-operator/cmd/pytorch-operator.v2/app/options"
	v1alpha2 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1alpha2"
	jobclientset "github.com/kubeflow/pytorch-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/pytorch-operator/pkg/util/testutil"
)

func TestAddPod(t *testing.T) {
	// Prepare the clientset and controller for the test.
	kubeClientSet := kubeclientset.NewForConfigOrDie(&rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &v1.SchemeGroupVersion,
		},
	},
	)
	config := &rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &v1alpha2.SchemeGroupVersion,
		},
	}
	jobClientSet := jobclientset.NewForConfigOrDie(config)
	ctr, _, _ := newPyTorchController(config, kubeClientSet, jobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
	ctr.jobInformerSynced = testutil.AlwaysReady
	ctr.PodInformerSynced = testutil.AlwaysReady
	ctr.ServiceInformerSynced = testutil.AlwaysReady
	jobIndexer := ctr.jobInformer.GetIndexer()

	stopCh := make(chan struct{})
	run := func(<-chan struct{}) {
		ctr.Run(testutil.ThreadCount, stopCh)
	}
	go run(stopCh)

	var key string
	syncChan := make(chan string)
	ctr.syncHandler = func(jobKey string) (bool, error) {
		key = jobKey
		<-syncChan
		return true, nil
	}

	job := testutil.NewPyTorchJobWithMaster(1)
	unstructured, err := testutil.ConvertPyTorchJobToUnstructured(job)
	if err != nil {
		t.Errorf("Failed to convert the job to Unstructured: %v", err)
	}

	if err := jobIndexer.Add(unstructured); err != nil {
		t.Errorf("Failed to add job to jobIndexer: %v", err)
	}
	pod := testutil.NewPod(job, testutil.LabelMaster, 0, t)
	ctr.AddPod(pod)

	syncChan <- "sync"
	if key != testutil.GetKey(job, t) {
		t.Errorf("Failed to enqueue the PyTorchJob %s: expected %s, got %s", job.Name, testutil.GetKey(job, t), key)
	}
	close(stopCh)
}

func TestClusterSpec(t *testing.T) {
	type tc struct {
		job                 *v1alpha2.PyTorchJob
		rt                  v1alpha2.PyTorchReplicaType
		index               string
		totalPods           int32
		expectedClusterSpec map[string]string
	}
	testCase := []tc{
		tc{
			job:                 testutil.NewPyTorchJobWithMaster(0),
			rt:                  v1alpha2.PyTorchReplicaTypeMaster,
			index:               "0",
			totalPods:           1,
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "1", "MASTER_PORT": "23456", "RANK": "0", "MASTER_ADDR": "localhost"},
		},
		tc{
			job:                 testutil.NewPyTorchJobWithMaster(1),
			rt:                  v1alpha2.PyTorchReplicaTypeMaster,
			index:               "0",
			totalPods:           2,
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "2", "MASTER_PORT": "23456", "RANK": "0", "MASTER_ADDR": "localhost"},
		},
		tc{
			job:                 testutil.NewPyTorchJobWithMaster(1),
			rt:                  v1alpha2.PyTorchReplicaTypeWorker,
			index:               "0",
			totalPods:           2,
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "2", "MASTER_PORT": "23456", "RANK": "1", "MASTER_ADDR": "test-pytorchjob-master-0"},
		},
		tc{
			job:                 testutil.NewPyTorchJobWithMaster(2),
			rt:                  v1alpha2.PyTorchReplicaTypeMaster,
			index:               "0",
			totalPods:           3,
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "3", "MASTER_PORT": "23456", "RANK": "0", "MASTER_ADDR": "localhost"},
		},
		tc{
			job:                 testutil.NewPyTorchJobWithMaster(2),
			rt:                  v1alpha2.PyTorchReplicaTypeWorker,
			index:               "0",
			totalPods:           3,
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "3", "MASTER_PORT": "23456", "RANK": "1", "MASTER_ADDR": "test-pytorchjob-master-0"},
		},
		tc{
			job:                 testutil.NewPyTorchJobWithMaster(2),
			rt:                  v1alpha2.PyTorchReplicaTypeWorker,
			index:               "1",
			totalPods:           3,
			expectedClusterSpec: map[string]string{"WORLD_SIZE": "3", "MASTER_PORT": "23456", "RANK": "2", "MASTER_ADDR": "test-pytorchjob-master-0"},
		},
	}
	for _, c := range testCase {
		demoTemplateSpec := c.job.Spec.PyTorchReplicaSpecs[c.rt].Template
		if err := setClusterSpec(&demoTemplateSpec, c.job, c.totalPods, c.index, c.rt); err != nil {
			t.Errorf("Failed to set cluster spec: %v", err)
		}
		actual := demoTemplateSpec.Spec.Containers[0].Env
		for _, env := range actual {
			if val, ok := c.expectedClusterSpec[env.Name]; ok {
				if val != env.Value {
					t.Errorf("For name %s Got %s. Expected %s ", env.Name, env.Value, c.expectedClusterSpec[env.Name])
				}
			}
		}
	}
}

func TestRestartPolicy(t *testing.T) {
	type tc struct {
		job                   *v1alpha2.PyTorchJob
		expectedRestartPolicy v1.RestartPolicy
		expectedType          v1alpha2.PyTorchReplicaType
	}
	testCase := []tc{
		func() tc {
			job := testutil.NewPyTorchJobWithMaster(1)
			specRestartPolicy := v1alpha2.RestartPolicyExitCode
			job.Spec.PyTorchReplicaSpecs[v1alpha2.PyTorchReplicaTypeMaster].RestartPolicy = specRestartPolicy
			return tc{
				job: job,
				expectedRestartPolicy: v1.RestartPolicyNever,
				expectedType:          v1alpha2.PyTorchReplicaTypeMaster,
			}
		}(),
		func() tc {
			job := testutil.NewPyTorchJobWithMaster(1)
			specRestartPolicy := v1alpha2.RestartPolicyNever
			job.Spec.PyTorchReplicaSpecs[v1alpha2.PyTorchReplicaTypeMaster].RestartPolicy = specRestartPolicy
			return tc{
				job: job,
				expectedRestartPolicy: v1.RestartPolicyNever,
				expectedType:          v1alpha2.PyTorchReplicaTypeMaster,
			}
		}(),
		func() tc {
			job := testutil.NewPyTorchJobWithMaster(1)
			specRestartPolicy := v1alpha2.RestartPolicyAlways
			job.Spec.PyTorchReplicaSpecs[v1alpha2.PyTorchReplicaTypeMaster].RestartPolicy = specRestartPolicy
			return tc{
				job: job,
				expectedRestartPolicy: v1.RestartPolicyAlways,
				expectedType:          v1alpha2.PyTorchReplicaTypeMaster,
			}
		}(),
		func() tc {
			job := testutil.NewPyTorchJobWithMaster(1)
			specRestartPolicy := v1alpha2.RestartPolicyOnFailure
			job.Spec.PyTorchReplicaSpecs[v1alpha2.PyTorchReplicaTypeMaster].RestartPolicy = specRestartPolicy
			return tc{
				job: job,
				expectedRestartPolicy: v1.RestartPolicyOnFailure,
				expectedType:          v1alpha2.PyTorchReplicaTypeMaster,
			}
		}(),
	}
	for _, c := range testCase {
		spec := c.job.Spec.PyTorchReplicaSpecs[c.expectedType]
		podTemplate := spec.Template
		setRestartPolicy(&podTemplate, spec)
		if podTemplate.Spec.RestartPolicy != c.expectedRestartPolicy {
			t.Errorf("Expected %s, got %s", c.expectedRestartPolicy, podTemplate.Spec.RestartPolicy)
		}
	}
}

func TestExitCode(t *testing.T) {
	// Prepare the clientset and controller for the test.
	kubeClientSet := kubeclientset.NewForConfigOrDie(&rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &v1.SchemeGroupVersion,
		},
	},
	)
	config := &rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &v1alpha2.SchemeGroupVersion,
		},
	}
	jobClientSet := jobclientset.NewForConfigOrDie(config)
	ctr, kubeInformerFactory, _ := newPyTorchController(config, kubeClientSet, jobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
	fakePodControl := &controller.FakePodControl{}
	ctr.PodControl = fakePodControl
	ctr.jobInformerSynced = testutil.AlwaysReady
	ctr.PodInformerSynced = testutil.AlwaysReady
	ctr.ServiceInformerSynced = testutil.AlwaysReady
	jobIndexer := ctr.jobInformer.GetIndexer()
	podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()

	stopCh := make(chan struct{})
	run := func(<-chan struct{}) {
		ctr.Run(testutil.ThreadCount, stopCh)
	}
	go run(stopCh)

	ctr.updateStatusHandler = func(job *v1alpha2.PyTorchJob) error {
		return nil
	}

	job := testutil.NewPyTorchJobWithMaster(1)
	job.Spec.PyTorchReplicaSpecs[v1alpha2.PyTorchReplicaTypeMaster].RestartPolicy = v1alpha2.RestartPolicyExitCode
	unstructured, err := testutil.ConvertPyTorchJobToUnstructured(job)
	if err != nil {
		t.Errorf("Failed to convert the PyTorchJob to Unstructured: %v", err)
	}

	if err := jobIndexer.Add(unstructured); err != nil {
		t.Errorf("Failed to add job to jobIndexer: %v", err)
	}
	pod := testutil.NewPod(job, testutil.LabelMaster, 0, t)
	pod.Status.Phase = v1.PodFailed
	pod.Spec.Containers = append(pod.Spec.Containers, v1.Container{})
	pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, v1.ContainerStatus{
		Name: v1alpha2.DefaultContainerName,
		State: v1.ContainerState{
			Terminated: &v1.ContainerStateTerminated{
				ExitCode: 130,
			},
		},
	})

	if err := podIndexer.Add(pod); err != nil {
		t.Errorf("%s: unexpected error when adding pod %v", job.Name, err)
	}
	_, err = ctr.syncPyTorchJob(testutil.GetKey(job, t))
	if err != nil {
		t.Errorf("%s: unexpected error when syncing jobs %v", job.Name, err)
	}

	found := false
	for _, deletedPodName := range fakePodControl.DeletePodName {
		if deletedPodName == pod.Name {
			found = true
		}
	}
	if !found {
		t.Errorf("Failed to delete pod %s", pod.Name)
	}
	close(stopCh)
}
