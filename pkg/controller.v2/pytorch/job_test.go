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
	"testing"
	"time"

	"k8s.io/api/core/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/controller"

	"github.com/kubeflow/pytorch-operator/cmd/pytorch-operator.v2/app/options"
	v1alpha2 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1alpha2"
	jobclientset "github.com/kubeflow/pytorch-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/pytorch-operator/pkg/util/testutil"
	"github.com/kubeflow/tf-operator/pkg/control"
)

func TestAddPyTorchJob(t *testing.T) {
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
	ctr.updateStatusHandler = func(job *v1alpha2.PyTorchJob) error {
		return nil
	}
	ctr.deletePyTorchJobHandler = func(job *v1alpha2.PyTorchJob) error {
		return nil
	}

	job := testutil.NewPyTorchJobWithMaster(1)
	unstructured, err := testutil.ConvertPyTorchJobToUnstructured(job)
	if err != nil {
		t.Errorf("Failed to convert the PyTorchJob to Unstructured: %v", err)
	}
	if err := jobIndexer.Add(unstructured); err != nil {
		t.Errorf("Failed to add job to jobIndexer: %v", err)
	}
	ctr.addPyTorchJob(unstructured)

	syncChan <- "sync"
	if key != testutil.GetKey(job, t) {
		t.Errorf("Failed to enqueue the PyTorchJob %s: expected %s, got %s", job.Name, testutil.GetKey(job, t), key)
	}
	close(stopCh)
}

func TestCopyLabelsAndAnnotation(t *testing.T) {
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
	fakePodControl := &controller.FakePodControl{}
	ctr.PodControl = fakePodControl
	ctr.jobInformerSynced = testutil.AlwaysReady
	ctr.PodInformerSynced = testutil.AlwaysReady
	ctr.ServiceInformerSynced = testutil.AlwaysReady
	jobIndexer := ctr.jobInformer.GetIndexer()

	stopCh := make(chan struct{})
	run := func(<-chan struct{}) {
		ctr.Run(testutil.ThreadCount, stopCh)
	}
	go run(stopCh)

	ctr.updateStatusHandler = func(job *v1alpha2.PyTorchJob) error {
		return nil
	}

	job := testutil.NewPyTorchJobWithMaster(0)
	annotations := map[string]string{
		"annotation1": "1",
	}
	labels := map[string]string{
		"label1": "1",
	}
	job.Spec.PyTorchReplicaSpecs[v1alpha2.PyTorchReplicaTypeMaster].Template.Labels = labels
	job.Spec.PyTorchReplicaSpecs[v1alpha2.PyTorchReplicaTypeMaster].Template.Annotations = annotations
	unstructured, err := testutil.ConvertPyTorchJobToUnstructured(job)
	if err != nil {
		t.Errorf("Failed to convert the PyTorchJob to Unstructured: %v", err)
	}

	if err := jobIndexer.Add(unstructured); err != nil {
		t.Errorf("Failed to add job to jobIndexer: %v", err)
	}

	_, err = ctr.syncPyTorchJob(testutil.GetKey(job, t))
	if err != nil {
		t.Errorf("%s: unexpected error when syncing jobs %v", job.Name, err)
	}

	if len(fakePodControl.Templates) != 1 {
		t.Errorf("Expected to create 1 pod while got %d", len(fakePodControl.Templates))
	}
	actual := fakePodControl.Templates[0]
	v, exist := actual.Labels["label1"]
	if !exist {
		t.Errorf("Labels does not exist")
	}
	if v != "1" {
		t.Errorf("Labels value do not equal")
	}

	v, exist = actual.Annotations["annotation1"]
	if !exist {
		t.Errorf("Annotations does not exist")
	}
	if v != "1" {
		t.Errorf("Annotations value does not equal")
	}

	close(stopCh)
}

func TestDeletePodsAndServices(t *testing.T) {
	type testCase struct {
		description string
		job         *v1alpha2.PyTorchJob

		pendingWorkerPods   int32
		activeWorkerPods    int32
		succeededWorkerPods int32
		failedWorkerPods    int32

		pendingMasterPods   int32
		activeMasterPods    int32
		succeededMasterPods int32
		failedMasterPods    int32

		activeWorkerServices int32
		activeMasterServices int32

		expectedPodDeletions int
	}

	testCases := []testCase{
		testCase{
			description: "4 workers and 1 master are running, policy is all",
			job:         testutil.NewPyTorchJobWithCleanPolicy(1, 4, v1alpha2.CleanPodPolicyAll),

			pendingWorkerPods:   0,
			activeWorkerPods:    4,
			succeededWorkerPods: 0,
			failedWorkerPods:    0,

			pendingMasterPods:   0,
			activeMasterPods:    1,
			succeededMasterPods: 0,
			failedMasterPods:    0,

			activeWorkerServices: 4,
			activeMasterServices: 1,

			expectedPodDeletions: 5,
		},
		testCase{
			description: "4 workers and 1 master succeeded, policy is None",
			job:         testutil.NewPyTorchJobWithCleanPolicy(1, 4, v1alpha2.CleanPodPolicyNone),

			pendingWorkerPods:   0,
			activeWorkerPods:    0,
			succeededWorkerPods: 4,
			failedWorkerPods:    0,

			pendingMasterPods:   0,
			activeMasterPods:    0,
			succeededMasterPods: 1,
			failedMasterPods:    0,

			activeWorkerServices: 4,
			activeMasterServices: 1,

			expectedPodDeletions: 0,
		},
	}
	for _, tc := range testCases {
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
		fakeServiceControl := &control.FakeServiceControl{}
		ctr.ServiceControl = fakeServiceControl
		ctr.Recorder = &record.FakeRecorder{}
		ctr.jobInformerSynced = testutil.AlwaysReady
		ctr.PodInformerSynced = testutil.AlwaysReady
		ctr.ServiceInformerSynced = testutil.AlwaysReady
		jobIndexer := ctr.jobInformer.GetIndexer()
		ctr.updateStatusHandler = func(job *v1alpha2.PyTorchJob) error {
			return nil
		}

		// Set succeeded to run the logic about deleting.
		err := updatePyTorchJobConditions(tc.job, v1alpha2.PyTorchJobSucceeded, pytorchJobSucceededReason, "")
		if err != nil {
			t.Errorf("Append job condition error: %v", err)
		}

		unstructured, err := testutil.ConvertPyTorchJobToUnstructured(tc.job)
		if err != nil {
			t.Errorf("Failed to convert the PyTorchJob to Unstructured: %v", err)
		}

		if err := jobIndexer.Add(unstructured); err != nil {
			t.Errorf("Failed to add job to jobIndexer: %v", err)
		}

		podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
		testutil.SetPodsStatuses(podIndexer, tc.job, testutil.LabelWorker, tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods, t)
		testutil.SetPodsStatuses(podIndexer, tc.job, testutil.LabelMaster, tc.pendingMasterPods, tc.activeMasterPods, tc.succeededMasterPods, tc.failedMasterPods, t)

		serviceIndexer := kubeInformerFactory.Core().V1().Services().Informer().GetIndexer()
		testutil.SetServices(serviceIndexer, tc.job, testutil.LabelWorker, tc.activeWorkerServices, t)
		testutil.SetServices(serviceIndexer, tc.job, testutil.LabelMaster, tc.activeMasterServices, t)

		forget, err := ctr.syncPyTorchJob(testutil.GetKey(tc.job, t))
		if err != nil {
			t.Errorf("%s: unexpected error when syncing jobs %v", tc.description, err)
		}
		if !forget {
			t.Errorf("%s: unexpected forget value. Expected true, saw %v\n", tc.description, forget)
		}

		if len(fakePodControl.DeletePodName) != tc.expectedPodDeletions {
			t.Errorf("%s: unexpected number of pod deletes.  Expected %d, saw %d\n", tc.description, tc.expectedPodDeletions, len(fakePodControl.DeletePodName))
		}
		if len(fakeServiceControl.DeleteServiceName) != tc.expectedPodDeletions {
			t.Errorf("%s: unexpected number of service deletes.  Expected %d, saw %d\n", tc.description, tc.expectedPodDeletions, len(fakeServiceControl.DeleteServiceName))
		}
	}
}

func TestCleanupPyTorchJob(t *testing.T) {
	type testCase struct {
		description string
		job         *v1alpha2.PyTorchJob

		pendingWorkerPods   int32
		activeWorkerPods    int32
		succeededWorkerPods int32
		failedWorkerPods    int32

		pendingMasterPods   int32
		activeMasterPods    int32
		succeededMasterPods int32
		failedMasterPods    int32

		activeWorkerServices int32
		activeMasterServices int32

		expectedDeleteFinished bool
	}

	ttlaf0 := int32(0)
	ttl0 := &ttlaf0
	ttlaf2s := int32(2)
	ttl2s := &ttlaf2s
	testCases := []testCase{
		testCase{
			description: "4 workers and 1 master are running, TTLSecondsAfterFinished unset",
			job:         testutil.NewPyTorchJobWithCleanupJobDelay(1, 4, nil),

			pendingWorkerPods:   0,
			activeWorkerPods:    4,
			succeededWorkerPods: 0,
			failedWorkerPods:    0,

			pendingMasterPods:   0,
			activeMasterPods:    1,
			succeededMasterPods: 0,
			failedMasterPods:    0,

			activeWorkerServices: 4,
			activeMasterServices: 1,

			expectedDeleteFinished: false,
		},
		testCase{
			description: "4 workers and 1 master are running, TTLSecondsAfterFinished is 0",
			job:         testutil.NewPyTorchJobWithCleanupJobDelay(1, 4, ttl0),

			pendingWorkerPods:   0,
			activeWorkerPods:    4,
			succeededWorkerPods: 0,
			failedWorkerPods:    0,

			pendingMasterPods:   0,
			activeMasterPods:    1,
			succeededMasterPods: 0,
			failedMasterPods:    0,

			activeWorkerServices: 4,
			activeMasterServices: 1,

			expectedDeleteFinished: true,
		},
		testCase{
			description: "4 workers and 1 master succeeded, TTLSecondsAfterFinished is 2",
			job:         testutil.NewPyTorchJobWithCleanupJobDelay(1, 4, ttl2s),

			pendingWorkerPods:   0,
			activeWorkerPods:    0,
			succeededWorkerPods: 4,
			failedWorkerPods:    0,

			pendingMasterPods:   0,
			activeMasterPods:    0,
			succeededMasterPods: 1,
			failedMasterPods:    0,

			activeWorkerServices: 4,
			activeMasterServices: 1,

			expectedDeleteFinished: true,
		},
	}
	for _, tc := range testCases {
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
		fakeServiceControl := &control.FakeServiceControl{}
		ctr.ServiceControl = fakeServiceControl
		ctr.Recorder = &record.FakeRecorder{}
		ctr.jobInformerSynced = testutil.AlwaysReady
		ctr.PodInformerSynced = testutil.AlwaysReady
		ctr.ServiceInformerSynced = testutil.AlwaysReady
		jobIndexer := ctr.jobInformer.GetIndexer()
		ctr.updateStatusHandler = func(job *v1alpha2.PyTorchJob) error {
			return nil
		}
		deleteFinished := false
		ctr.deletePyTorchJobHandler = func(job *v1alpha2.PyTorchJob) error {
			deleteFinished = true
			return nil
		}

		// Set succeeded to run the logic about deleting.
		testutil.SetPyTorchJobCompletionTime(tc.job)

		err := updatePyTorchJobConditions(tc.job, v1alpha2.PyTorchJobSucceeded, pytorchJobSucceededReason, "")
		if err != nil {
			t.Errorf("Append job condition error: %v", err)
		}

		unstructured, err := testutil.ConvertPyTorchJobToUnstructured(tc.job)
		if err != nil {
			t.Errorf("Failed to convert the PyTorchJob to Unstructured: %v", err)
		}

		if err := jobIndexer.Add(unstructured); err != nil {
			t.Errorf("Failed to add job to jobIndexer: %v", err)
		}

		podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
		testutil.SetPodsStatuses(podIndexer, tc.job, testutil.LabelWorker, tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods, t)
		testutil.SetPodsStatuses(podIndexer, tc.job, testutil.LabelMaster, tc.pendingMasterPods, tc.activeMasterPods, tc.succeededMasterPods, tc.failedMasterPods, t)

		serviceIndexer := kubeInformerFactory.Core().V1().Services().Informer().GetIndexer()
		testutil.SetServices(serviceIndexer, tc.job, testutil.LabelWorker, tc.activeWorkerServices, t)
		testutil.SetServices(serviceIndexer, tc.job, testutil.LabelMaster, tc.activeMasterServices, t)

		ttl := tc.job.Spec.TTLSecondsAfterFinished
		if ttl != nil {
			dur := time.Second * time.Duration(*ttl)
			time.Sleep(dur)
		}

		forget, err := ctr.syncPyTorchJob(testutil.GetKey(tc.job, t))
		if err != nil {
			t.Errorf("%s: unexpected error when syncing jobs %v", tc.description, err)
		}
		if !forget {
			t.Errorf("%s: unexpected forget value. Expected true, saw %v\n", tc.description, forget)
		}

		if deleteFinished != tc.expectedDeleteFinished {
			t.Errorf("%s: unexpected status. Expected %v, saw %v", tc.description, tc.expectedDeleteFinished, deleteFinished)
		}
	}
}
