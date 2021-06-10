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
	"time"

	kubebatchclient "github.com/kubernetes-sigs/kube-batch/pkg/client/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/controller"

	common "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/pytorch-operator/cmd/pytorch-operator.v1/app/options"
	pyv1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1"
	jobclientset "github.com/kubeflow/pytorch-operator/pkg/client/clientset/versioned"
	jobinformers "github.com/kubeflow/pytorch-operator/pkg/client/informers/externalversions"
	"github.com/kubeflow/pytorch-operator/pkg/common/util/v1/testutil"
	"github.com/kubeflow/tf-operator/pkg/control"
)

var (
	jobRunning   = common.JobRunning
	jobSucceeded = common.JobSucceeded
)

func newPyTorchController(
	config *rest.Config,
	kubeClientSet kubeclientset.Interface,
	kubeBatchClientSet kubebatchclient.Interface,
	jobClientSet jobclientset.Interface,
	resyncPeriod controller.ResyncPeriodFunc,
	option options.ServerOption,
) (
	*PyTorchController,
	kubeinformers.SharedInformerFactory, jobinformers.SharedInformerFactory,
) {
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClientSet, resyncPeriod())
	jobInformerFactory := jobinformers.NewSharedInformerFactory(jobClientSet, resyncPeriod())

	jobInformer := NewUnstructuredPyTorchJobInformer(config, metav1.NamespaceAll)

	ctr := NewPyTorchController(jobInformer, kubeClientSet, kubeBatchClientSet, jobClientSet, kubeInformerFactory, jobInformerFactory, option)
	ctr.PodControl = &controller.FakePodControl{}
	ctr.ServiceControl = &control.FakeServiceControl{}
	return ctr, kubeInformerFactory, jobInformerFactory
}

func TestNormalPath(t *testing.T) {
	testCases := map[string]struct {
		worker int
		master int

		// pod setup
		ControllerError error
		jobKeyForget    bool

		pendingWorkerPods   int32
		activeWorkerPods    int32
		succeededWorkerPods int32
		failedWorkerPods    int32

		pendingMasterPods   int32
		activeMasterPods    int32
		succeededMasterPods int32
		failedMasterPods    int32

		activeMasterServices int32

		// expectations
		expectedPodCreations     int32
		expectedPodDeletions     int32
		expectedServiceCreations int32

		expectedActiveWorkerPods    int32
		expectedSucceededWorkerPods int32
		expectedFailedWorkerPods    int32

		expectedActiveMasterPods    int32
		expectedSucceededMasterPods int32
		expectedFailedMasterPods    int32

		expectedCondition       *common.JobConditionType
		expectedConditionReason string

		// There are some cases that should not check start time since the field should be set in the previous sync loop.
		needCheckStartTime bool
	}{
		"Local PyTorchJob is created": {
			0, 1,
			nil, true,
			0, 0, 0, 0,
			0, 0, 0, 0,
			0,
			1, 0, 1,
			0, 0, 0,
			0, 0, 0,
			// We can not check if it is created since the condition is set in addPyTorchJob.
			nil, "",
			false,
		},
		"Distributed PyTorchJob (4 workers, 1 master) is created": {
			4, 1,
			nil, true,
			0, 0, 0, 0,
			0, 0, 0, 0,
			0,
			5, 0, 1,
			0, 0, 0,
			0, 0, 0,
			nil, "",
			false,
		},
		"Distributed PyTorchJob (4 workers, 1 master) is created, 1 master and 4 workers are pending": {
			4, 1,
			nil, true,
			4, 0, 0, 0,
			1, 0, 0, 0,
			1,
			0, 0, 0,
			0, 0, 0,
			0, 0, 0,
			nil, "",
			false,
		},
		"Distributed PyTorchJob (4 workers, 1 master) is created, 2 workers pending, 1 master 1 worker are running": {
			4, 1,
			nil, true,
			3, 1, 0, 0,
			0, 1, 0, 0,
			1,
			0, 0, 0,
			1, 0, 0,
			1, 0, 0,
			&jobRunning, pytorchJobRunningReason,
			false,
		},
		"Distributed PyTorchJob (4 workers, 1 master) is created and all replicas are running": {
			4, 1,
			nil, true,
			0, 4, 0, 0,
			0, 1, 0, 0,
			1,
			0, 0, 0,
			4, 0, 0,
			1, 0, 0,
			&jobRunning, pytorchJobRunningReason,
			true,
		},
		"Distributed PyTorchJob (4 workers, 1 master) is succeeded": {
			4, 1,
			nil, true,
			0, 0, 4, 0,
			0, 0, 1, 0,
			1,
			0, 0, 0,
			0, 4, 0,
			0, 1, 0,
			&jobSucceeded, pytorchJobSucceededReason,
			false,
		},
	}

	for name, tc := range testCases {
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
		option := options.ServerOption{}
		jobClientSet := jobclientset.NewForConfigOrDie(config)
		ctr, kubeInformerFactory, _ := newPyTorchController(config, kubeClientSet, kubeBatchClientSet, jobClientSet, controller.NoResyncPeriodFunc, option)
		ctr.jobInformerSynced = testutil.AlwaysReady
		ctr.PodInformerSynced = testutil.AlwaysReady
		ctr.ServiceInformerSynced = testutil.AlwaysReady
		jobIndexer := ctr.jobInformer.GetIndexer()

		var actual *pyv1.PyTorchJob
		ctr.updateStatusHandler = func(job *pyv1.PyTorchJob) error {
			actual = job
			return nil
		}

		// Run the test logic.
		job := testutil.NewPyTorchJobWithMaster(tc.worker)
		unstructured, err := testutil.ConvertPyTorchJobToUnstructured(job)
		if err != nil {
			t.Errorf("Failed to convert the PyTorchJob to Unstructured: %v", err)
		}

		if err := jobIndexer.Add(unstructured); err != nil {
			t.Errorf("Failed to add job to jobIndexer: %v", err)
		}

		podIndexer := kubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
		testutil.SetPodsStatuses(podIndexer, job, testutil.LabelWorker, tc.pendingWorkerPods, tc.activeWorkerPods, tc.succeededWorkerPods, tc.failedWorkerPods, nil, t)
		testutil.SetPodsStatuses(podIndexer, job, testutil.LabelMaster, tc.pendingMasterPods, tc.activeMasterPods, tc.succeededMasterPods, tc.failedMasterPods, nil, t)

		serviceIndexer := kubeInformerFactory.Core().V1().Services().Informer().GetIndexer()
		testutil.SetServices(serviceIndexer, job, testutil.LabelMaster, tc.activeMasterServices, t)

		forget, err := ctr.syncPyTorchJob(testutil.GetKey(job, t))
		// We need requeue syncJob task if podController error
		if tc.ControllerError != nil {
			if err == nil {
				t.Errorf("%s: Syncing jobs would return error when podController exception", name)
			}
		} else {
			if err != nil {
				t.Errorf("%s: unexpected error when syncing jobs %v", name, err)
			}
		}
		if forget != tc.jobKeyForget {
			t.Errorf("%s: unexpected forget value. Expected %v, saw %v\n", name, tc.jobKeyForget, forget)
		}

		fakePodControl := ctr.PodControl.(*controller.FakePodControl)
		fakeServiceControl := ctr.ServiceControl.(*control.FakeServiceControl)
		if int32(len(fakePodControl.Templates)) != tc.expectedPodCreations {
			t.Errorf("%s: unexpected number of pod creates.  Expected %d, saw %d\n", name, tc.expectedPodCreations, len(fakePodControl.Templates))
		}
		if int32(len(fakeServiceControl.Templates)) != tc.expectedServiceCreations {
			t.Errorf("%s: unexpected number of service creates.  Expected %d, saw %d\n", name, tc.expectedServiceCreations, len(fakeServiceControl.Templates))
		}
		if int32(len(fakePodControl.DeletePodName)) != tc.expectedPodDeletions {
			t.Errorf("%s: unexpected number of pod deletes.  Expected %d, saw %d\n", name, tc.expectedPodDeletions, len(fakePodControl.DeletePodName))
		}
		// Each create should have an accompanying ControllerRef.
		if len(fakePodControl.ControllerRefs) != int(tc.expectedPodCreations) {
			t.Errorf("%s: unexpected number of ControllerRefs.  Expected %d, saw %d\n", name, tc.expectedPodCreations, len(fakePodControl.ControllerRefs))
		}
		// Make sure the ControllerRefs are correct.
		for _, controllerRef := range fakePodControl.ControllerRefs {
			if got, want := controllerRef.APIVersion, pyv1.SchemeGroupVersion.String(); got != want {
				t.Errorf("controllerRef.APIVersion = %q, want %q", got, want)
			}
			if got, want := controllerRef.Kind, pyv1.Kind; got != want {
				t.Errorf("controllerRef.Kind = %q, want %q", got, want)
			}
			if got, want := controllerRef.Name, job.Name; got != want {
				t.Errorf("controllerRef.Name = %q, want %q", got, want)
			}
			if got, want := controllerRef.UID, job.UID; got != want {
				t.Errorf("controllerRef.UID = %q, want %q", got, want)
			}
			if controllerRef.Controller == nil || !*controllerRef.Controller {
				t.Errorf("controllerRef.Controller is not set to true")
			}
		}
		// Validate worker status.
		if actual.Status.ReplicaStatuses[common.ReplicaType(pyv1.PyTorchReplicaTypeWorker)] != nil {
			if actual.Status.ReplicaStatuses[common.ReplicaType(pyv1.PyTorchReplicaTypeWorker)].Active != tc.expectedActiveWorkerPods {
				t.Errorf("%s: unexpected number of active pods.  Expected %d, saw %d\n", name, tc.expectedActiveWorkerPods, actual.Status.ReplicaStatuses[common.ReplicaType(pyv1.PyTorchReplicaTypeWorker)].Active)
			}
			if actual.Status.ReplicaStatuses[common.ReplicaType(pyv1.PyTorchReplicaTypeWorker)].Succeeded != tc.expectedSucceededWorkerPods {
				t.Errorf("%s: unexpected number of succeeded pods.  Expected %d, saw %d\n", name, tc.expectedSucceededWorkerPods, actual.Status.ReplicaStatuses[common.ReplicaType(pyv1.PyTorchReplicaTypeWorker)].Succeeded)
			}
			if actual.Status.ReplicaStatuses[common.ReplicaType(pyv1.PyTorchReplicaTypeWorker)].Failed != tc.expectedFailedWorkerPods {
				t.Errorf("%s: unexpected number of failed pods.  Expected %d, saw %d\n", name, tc.expectedFailedWorkerPods, actual.Status.ReplicaStatuses[common.ReplicaType(pyv1.PyTorchReplicaTypeWorker)].Failed)
			}
		}

		// Validate StartTime.
		if tc.needCheckStartTime && actual.Status.StartTime == nil {
			t.Errorf("%s: StartTime was not set", name)
		}
		// Validate conditions.
		if tc.expectedCondition != nil && !testutil.CheckCondition(actual, *tc.expectedCondition, tc.expectedConditionReason) {
			t.Errorf("%s: expected condition %#v, got %#v", name, *tc.expectedCondition, actual.Status.Conditions)
		}
	}
}

func TestRun(t *testing.T) {
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
		t.Errorf("Failed to run: %v", err)
	}
}
