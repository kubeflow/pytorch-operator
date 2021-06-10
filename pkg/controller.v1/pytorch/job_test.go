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

	kubebatchclient "github.com/kubernetes-sigs/kube-batch/pkg/client/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/controller"

	common "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/pytorch-operator/cmd/pytorch-operator.v1/app/options"
	pyv1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1"
	jobclientset "github.com/kubeflow/pytorch-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/pytorch-operator/pkg/common/util/v1/testutil"
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
	// Prepare the kube-batch clientset and controller for the test.
	kubeBatchClientSet := kubebatchclient.NewForConfigOrDie(&rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &v1.SchemeGroupVersion,
		},
	},
	)

	config := &rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &pyv1.SchemeGroupVersion,
		},
	}
	jobClientSet := jobclientset.NewForConfigOrDie(config)
	ctr, _, _ := newPyTorchController(config, kubeClientSet, kubeBatchClientSet, jobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
	ctr.jobInformerSynced = testutil.AlwaysReady
	ctr.PodInformerSynced = testutil.AlwaysReady
	ctr.ServiceInformerSynced = testutil.AlwaysReady
	jobIndexer := ctr.jobInformer.GetIndexer()

	stopCh := make(chan struct{})
	run := func(<-chan struct{}) {
		if err := ctr.Run(testutil.ThreadCount, stopCh); err != nil {
			t.Errorf("Failed to run the controller: %v", err)
		}
	}
	go run(stopCh)

	var key string
	syncChan := make(chan string)
	ctr.syncHandler = func(jobKey string) (bool, error) {
		key = jobKey
		<-syncChan
		return true, nil
	}
	ctr.updateStatusHandler = func(job *pyv1.PyTorchJob) error {
		return nil
	}
	ctr.deletePyTorchJobHandler = func(job *pyv1.PyTorchJob) error {
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
	// Prepare the kube-batch clientset and controller for the test.
	kubeBatchClientSet := kubebatchclient.NewForConfigOrDie(&rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &v1.SchemeGroupVersion,
		},
	},
	)

	config := &rest.Config{
		Host: "",
		ContentConfig: rest.ContentConfig{
			GroupVersion: &pyv1.SchemeGroupVersion,
		},
	}
	jobClientSet := jobclientset.NewForConfigOrDie(config)
	ctr, _, _ := newPyTorchController(config, kubeClientSet, kubeBatchClientSet, jobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
	fakePodControl := &controller.FakePodControl{}
	ctr.PodControl = fakePodControl
	ctr.jobInformerSynced = testutil.AlwaysReady
	ctr.PodInformerSynced = testutil.AlwaysReady
	ctr.ServiceInformerSynced = testutil.AlwaysReady
	jobIndexer := ctr.jobInformer.GetIndexer()

	stopCh := make(chan struct{})

	go func() {
		// It is a hack to let the controller stop to run without errors.
		// We can not just send a struct to stopCh because there are multiple
		// receivers in controller.Run.
		time.Sleep(testutil.SleepInterval)
		stopCh <- struct{}{}
	}()
	err := ctr.Run(testutil.ThreadCount, stopCh)
	if err != nil {
		t.Errorf("Failed to run the controller: %v", err)
	}

	ctr.updateStatusHandler = func(job *pyv1.PyTorchJob) error {
		return nil
	}

	job := testutil.NewPyTorchJobWithMaster(0)
	annotations := map[string]string{
		"annotation1": "1",
	}
	labels := map[string]string{
		"label1": "1",
	}
	job.Spec.PyTorchReplicaSpecs[pyv1.PyTorchReplicaTypeMaster].Template.Labels = labels
	job.Spec.PyTorchReplicaSpecs[pyv1.PyTorchReplicaTypeMaster].Template.Annotations = annotations
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
		job         *pyv1.PyTorchJob

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

		expectedPodDeletions     int
		expectedServiceDeletions int
	}

	testCases := []testCase{
		testCase{
			description: "4 workers and 1 master are running, policy is all",
			job:         testutil.NewPyTorchJobWithCleanPolicy(1, 4, common.CleanPodPolicyAll),

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

			expectedPodDeletions:     5,
			expectedServiceDeletions: 1,
		},
		testCase{
			description: "4 workers and 1 master running, policy is running",
			job:         testutil.NewPyTorchJobWithCleanPolicy(1, 4, common.CleanPodPolicyRunning),

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

			expectedPodDeletions:     5,
			expectedServiceDeletions: 1,
		},
		testCase{
			description: "4 workers and 1 master succeeded, policy is running",
			job:         testutil.NewPyTorchJobWithCleanPolicy(1, 4, common.CleanPodPolicyRunning),

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

			expectedPodDeletions:     0,
			expectedServiceDeletions: 1,
		},
		testCase{
			description: "4 workers and 1 master succeeded, policy is None",
			job:         testutil.NewPyTorchJobWithCleanPolicy(1, 4, common.CleanPodPolicyNone),

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

			expectedPodDeletions:     0,
			expectedServiceDeletions: 0,
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
		// Prepare the kube-batch clientset and controller for the test.
		kubeBatchClientSet := kubebatchclient.NewForConfigOrDie(&rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &v1.SchemeGroupVersion,
			},
		},
		)

		config := &rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &pyv1.SchemeGroupVersion,
			},
		}
		jobClientSet := jobclientset.NewForConfigOrDie(config)
		ctr, kubeInformerFactory, _ := newPyTorchController(config, kubeClientSet, kubeBatchClientSet, jobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
		fakePodControl := &controller.FakePodControl{}
		ctr.PodControl = fakePodControl
		fakeServiceControl := &control.FakeServiceControl{}
		ctr.ServiceControl = fakeServiceControl
		ctr.Recorder = &record.FakeRecorder{}
		ctr.jobInformerSynced = testutil.AlwaysReady
		ctr.PodInformerSynced = testutil.AlwaysReady
		ctr.ServiceInformerSynced = testutil.AlwaysReady
		jobIndexer := ctr.jobInformer.GetIndexer()
		ctr.updateStatusHandler = func(job *pyv1.PyTorchJob) error {
			return nil
		}

		// Set succeeded to run the logic about deleting.
		err := updatePyTorchJobConditions(tc.job, common.JobSucceeded, pytorchJobSucceededReason, "")
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
		testutil.SetPodsStatuses(podIndexer, tc.job, testutil.LabelWorker, tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods, nil, t)
		testutil.SetPodsStatuses(podIndexer, tc.job, testutil.LabelMaster, tc.pendingMasterPods, tc.activeMasterPods, tc.succeededMasterPods, tc.failedMasterPods, nil, t)

		serviceIndexer := kubeInformerFactory.Core().V1().Services().Informer().GetIndexer()
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
		if len(fakeServiceControl.DeleteServiceName) != tc.expectedServiceDeletions {
			t.Errorf("%s: unexpected number of service deletes.  Expected %d, saw %d\n", tc.description, tc.expectedServiceDeletions, len(fakeServiceControl.DeleteServiceName))
		}
	}
}

func TestCleanupPyTorchJob(t *testing.T) {
	type testCase struct {
		description string
		job         *pyv1.PyTorchJob

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
		// Prepare the kube-batch clientset and controller for the test.
		kubeBatchClientSet := kubebatchclient.NewForConfigOrDie(&rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &v1.SchemeGroupVersion,
			},
		},
		)

		config := &rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &pyv1.SchemeGroupVersion,
			},
		}
		jobClientSet := jobclientset.NewForConfigOrDie(config)
		ctr, kubeInformerFactory, _ := newPyTorchController(config, kubeClientSet, kubeBatchClientSet, jobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
		fakePodControl := &controller.FakePodControl{}
		ctr.PodControl = fakePodControl
		fakeServiceControl := &control.FakeServiceControl{}
		ctr.ServiceControl = fakeServiceControl
		ctr.Recorder = &record.FakeRecorder{}
		ctr.jobInformerSynced = testutil.AlwaysReady
		ctr.PodInformerSynced = testutil.AlwaysReady
		ctr.ServiceInformerSynced = testutil.AlwaysReady
		jobIndexer := ctr.jobInformer.GetIndexer()
		ctr.updateStatusHandler = func(job *pyv1.PyTorchJob) error {
			return nil
		}
		deleteFinished := false
		ctr.deletePyTorchJobHandler = func(job *pyv1.PyTorchJob) error {
			deleteFinished = true
			return nil
		}

		// Set succeeded to run the logic about deleting.
		testutil.SetPyTorchJobCompletionTime(tc.job)

		err := updatePyTorchJobConditions(tc.job, common.JobSucceeded, pytorchJobSucceededReason, "")
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
		testutil.SetPodsStatuses(podIndexer, tc.job, testutil.LabelWorker, tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods, nil, t)
		testutil.SetPodsStatuses(podIndexer, tc.job, testutil.LabelMaster, tc.pendingMasterPods, tc.activeMasterPods, tc.succeededMasterPods, tc.failedMasterPods, nil, t)

		serviceIndexer := kubeInformerFactory.Core().V1().Services().Informer().GetIndexer()
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

func TestActiveDeadlineSeconds(t *testing.T) {
	type testCase struct {
		description string
		job         *pyv1.PyTorchJob

		pendingWorkerPods   int32
		activeWorkerPods    int32
		succeededWorkerPods int32
		failedWorkerPods    int32

		pendingMasterPods   int32
		activeMasterPods    int32
		succeededMasterPods int32
		failedMasterPods    int32

		activeMasterServices int32

		expectedPodDeletions     int
		expectedServiceDeletions int
	}

	ads2 := int64(2)
	adsTest2 := &ads2
	testCases := []testCase{
		testCase{
			description: "1 master and 4 workers running, ActiveDeadlineSeconds unset",
			job:         testutil.NewPyTorchJobWithActiveDeadlineSeconds(1, 4, nil),

			pendingWorkerPods:   0,
			activeWorkerPods:    4,
			succeededWorkerPods: 0,
			failedWorkerPods:    0,

			pendingMasterPods:   0,
			activeMasterPods:    1,
			succeededMasterPods: 0,
			failedMasterPods:    0,

			activeMasterServices: 1,

			expectedPodDeletions:     0,
			expectedServiceDeletions: 0,
		},
		testCase{
			description: "1 master and 4 workers running, ActiveDeadlineSeconds is 2",
			job:         testutil.NewPyTorchJobWithActiveDeadlineSeconds(1, 4, adsTest2),

			pendingWorkerPods:   0,
			activeWorkerPods:    4,
			succeededWorkerPods: 0,
			failedWorkerPods:    0,

			pendingMasterPods:   0,
			activeMasterPods:    1,
			succeededMasterPods: 0,
			failedMasterPods:    0,

			activeMasterServices: 1,

			expectedPodDeletions:     5,
			expectedServiceDeletions: 1,
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

		// Prepare the kube-batch clientset and controller for the test.
		kubeBatchClientSet := kubebatchclient.NewForConfigOrDie(&rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &v1.SchemeGroupVersion,
			},
		},
		)

		config := &rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &pyv1.SchemeGroupVersion,
			},
		}
		jobClientSet := jobclientset.NewForConfigOrDie(config)
		ctr, kubeInformerFactory, _ := newPyTorchController(config, kubeClientSet, kubeBatchClientSet, jobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
		fakePodControl := &controller.FakePodControl{}
		ctr.PodControl = fakePodControl
		fakeServiceControl := &control.FakeServiceControl{}
		ctr.ServiceControl = fakeServiceControl
		ctr.Recorder = &record.FakeRecorder{}
		ctr.jobInformerSynced = testutil.AlwaysReady
		ctr.PodInformerSynced = testutil.AlwaysReady
		ctr.ServiceInformerSynced = testutil.AlwaysReady
		jobIndexer := ctr.jobInformer.GetIndexer()
		ctr.updateStatusHandler = func(job *pyv1.PyTorchJob) error {
			return nil
		}

		unstructured, err := testutil.ConvertPyTorchJobToUnstructured(tc.job)
		if err != nil {
			t.Errorf("Failed to convert the PyTorchJob to Unstructured: %v", err)
		}

		if err := jobIndexer.Add(unstructured); err != nil {
			t.Errorf("Failed to add job to jobIndexer: %v", err)
		}

		podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
		testutil.SetPodsStatuses(podIndexer, tc.job, testutil.LabelWorker, tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods, nil, t)
		testutil.SetPodsStatuses(podIndexer, tc.job, testutil.LabelMaster, tc.pendingMasterPods, tc.activeMasterPods, tc.succeededMasterPods, tc.failedMasterPods, nil, t)

		serviceIndexer := kubeInformerFactory.Core().V1().Services().Informer().GetIndexer()

		testutil.SetServices(serviceIndexer, tc.job, testutil.LabelMaster, tc.activeMasterServices, t)

		foo, _ := ctr.getPyTorchJobFromName("default", "test-pytorchjob")
		now := metav1.Now()
		foo.Status.StartTime = &now

		ads := tc.job.Spec.ActiveDeadlineSeconds
		if ads != nil {
			dur := time.Second * time.Duration(*ads)
			time.Sleep(dur)
		}
		err = ctr.reconcilePyTorchJobs(foo)
		if err != nil {
			t.Errorf("%s: unexpected error when syncing jobs %v", tc.description, err)
		}

		if len(fakePodControl.DeletePodName) != tc.expectedPodDeletions {
			t.Errorf("%s: unexpected number of pod deletes.  Expected %d, saw %d\n", tc.description, tc.expectedPodDeletions, len(fakePodControl.DeletePodName))
		}
		if len(fakeServiceControl.DeleteServiceName) != tc.expectedServiceDeletions {
			t.Errorf("%s: unexpected number of service deletes.  Expected %d, saw %d\n", tc.description, tc.expectedServiceDeletions, len(fakeServiceControl.DeleteServiceName))
		}
	}
}

func TestBackoffForOnFailure(t *testing.T) {
	type testCase struct {
		description string
		job         *pyv1.PyTorchJob

		pendingWorkerPods   int32
		activeWorkerPods    int32
		succeededWorkerPods int32
		failedWorkerPods    int32

		restartCounts []int32

		pendingMasterPods   int32
		activeMasterPods    int32
		succeededMasterPods int32
		failedMasterPods    int32

		activeMasterServices int32

		expectedPodDeletions     int
		expectedServiceDeletions int
	}

	backoffLimit4 := int32(4)
	backoffLimitTest4 := &backoffLimit4
	testCases := []testCase{
		testCase{
			description: "1 master and 4 workers each having 1 restartCount running, backoffLimit 4 ",
			job:         testutil.NewPyTorchJobWithBackoffLimit(1, 4, backoffLimitTest4),

			pendingWorkerPods:   0,
			activeWorkerPods:    4,
			succeededWorkerPods: 0,
			failedWorkerPods:    0,

			restartCounts: []int32{1, 1, 1, 1},

			pendingMasterPods:   0,
			activeMasterPods:    1,
			succeededMasterPods: 0,
			failedMasterPods:    0,

			activeMasterServices: 1,

			expectedPodDeletions:     5,
			expectedServiceDeletions: 1,
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

		// Prepare the kube-batch clientset and controller for the test.
		kubeBatchClientSet := kubebatchclient.NewForConfigOrDie(&rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &v1.SchemeGroupVersion,
			},
		},
		)

		config := &rest.Config{
			Host: "",
			ContentConfig: rest.ContentConfig{
				GroupVersion: &pyv1.SchemeGroupVersion,
			},
		}
		jobClientSet := jobclientset.NewForConfigOrDie(config)
		ctr, kubeInformerFactory, _ := newPyTorchController(config, kubeClientSet, kubeBatchClientSet, jobClientSet, controller.NoResyncPeriodFunc, options.ServerOption{})
		fakePodControl := &controller.FakePodControl{}
		ctr.PodControl = fakePodControl
		fakeServiceControl := &control.FakeServiceControl{}
		ctr.ServiceControl = fakeServiceControl
		ctr.Recorder = &record.FakeRecorder{}
		ctr.jobInformerSynced = testutil.AlwaysReady
		ctr.PodInformerSynced = testutil.AlwaysReady
		ctr.ServiceInformerSynced = testutil.AlwaysReady
		jobIndexer := ctr.jobInformer.GetIndexer()
		ctr.updateStatusHandler = func(job *pyv1.PyTorchJob) error {
			return nil
		}

		unstructured, err := testutil.ConvertPyTorchJobToUnstructured(tc.job)
		if err != nil {
			t.Errorf("Failed to convert the PyTorchJob to Unstructured: %v", err)
		}

		if err := jobIndexer.Add(unstructured); err != nil {
			t.Errorf("Failed to add job to jobIndexer: %v", err)
		}

		podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
		testutil.SetPodsStatuses(podIndexer, tc.job, testutil.LabelWorker, tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods, tc.restartCounts, t)
		testutil.SetPodsStatuses(podIndexer, tc.job, testutil.LabelMaster, tc.pendingMasterPods, tc.activeMasterPods, tc.succeededMasterPods, tc.failedMasterPods, tc.restartCounts, t)

		serviceIndexer := kubeInformerFactory.Core().V1().Services().Informer().GetIndexer()

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
		if len(fakeServiceControl.DeleteServiceName) != tc.expectedServiceDeletions {
			t.Errorf("%s: unexpected number of service deletes.  Expected %d, saw %d\n", tc.description, tc.expectedServiceDeletions, len(fakeServiceControl.DeleteServiceName))
		}
	}
}
