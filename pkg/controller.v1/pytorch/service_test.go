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

	kubebatchclient "github.com/kubernetes-sigs/kube-batch/pkg/client/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/controller"

	"github.com/kubeflow/pytorch-operator/cmd/pytorch-operator.v1/app/options"
	pyv1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1"
	jobclientset "github.com/kubeflow/pytorch-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/pytorch-operator/pkg/common/util/v1/testutil"
)

func TestAddService(t *testing.T) {
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

	job := testutil.NewPyTorchJobWithMaster(1)
	unstructured, err := testutil.ConvertPyTorchJobToUnstructured(job)
	if err != nil {
		t.Errorf("Failed to convert the PyTorchJob to Unstructured: %v", err)
	}

	if err := jobIndexer.Add(unstructured); err != nil {
		t.Errorf("Failed to add tfjob to jobIndexer: %v", err)
	}
	service := testutil.NewService(job, testutil.LabelMaster, 0, t)
	ctr.AddService(service)

	syncChan <- "sync"
	if key != testutil.GetKey(job, t) {
		t.Errorf("Failed to enqueue the PyTorchJob %s: expected %s, got %s", job.Name, testutil.GetKey(job, t), key)
	}
	close(stopCh)
}
