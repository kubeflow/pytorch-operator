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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	pyv1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1"
	"github.com/kubeflow/pytorch-operator/pkg/common/util/v1/testutil"
	"github.com/kubeflow/tf-operator/pkg/common/jobcontroller"
)

func TestGenOwnerReference(t *testing.T) {
	testName := "test-job"
	testUID := types.UID("test-UID")
	job := &pyv1.PyTorchJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: testName,
			UID:  testUID,
		},
	}

	ref := testutil.GenOwnerReference(job)
	if ref.UID != testUID {
		t.Errorf("Expected UID %s, got %s", testUID, ref.UID)
	}
	if ref.Name != testName {
		t.Errorf("Expected Name %s, got %s", testName, ref.Name)
	}
	if ref.APIVersion != pyv1.SchemeGroupVersion.String() {
		t.Errorf("Expected APIVersion %s, got %s", pyv1.SchemeGroupVersion.String(), ref.APIVersion)
	}
}

func TestGenLabels(t *testing.T) {
	testKey := "test/key"
	expectedKey := "test-key"

	labels := testutil.GenLabels(testKey)
	jobNamelabel := jobcontroller.JobNameLabel

	controllerName := jobcontroller.ControllerNameLabel
	expectedcontrollerName := "pytorch-operator"

	if labels[jobNamelabel] != expectedKey {
		t.Errorf("Expected %s %s, got %s", jobNamelabel, expectedKey, labels[jobNamelabel])
	}
	if labels[labelGroupName] != pyv1.GroupName {
		t.Errorf("Expected %s %s, got %s", labelGroupName, pyv1.GroupName, labels[labelGroupName])
	}
	if labels[controllerName] != expectedcontrollerName {
		t.Errorf("Expected %s %s, got %s", controllerName, expectedcontrollerName, labels[controllerName])
	}

}

func TestConvertPyTorchJobToUnstructured(t *testing.T) {
	testName := "test-job"
	testUID := types.UID("test-UID")
	job := &pyv1.PyTorchJob{
		TypeMeta: metav1.TypeMeta{
			Kind: pyv1.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: testName,
			UID:  testUID,
		},
	}

	_, err := testutil.ConvertPyTorchJobToUnstructured(job)
	if err != nil {
		t.Errorf("Expected error to be nil while got %v", err)
	}
}

func TestGetInitContainer(t *testing.T) {
	template := `
- name: init-pytorch
  image: busybox
  command: ['sh', '-c', 'until nslookup {{.MasterAddr}}; do echo waiting for master; sleep 2; done;']`

	initContainer, err := GetInitContainer(template, InitContainerParam{
		MasterAddr:         "svc",
		InitContainerImage: "busybox",
	})
	if err != nil {
		t.Errorf("Expected error to be nil while got %v", err)
	}

	expectedCMD := "until nslookup svc; do echo waiting for master; sleep 2; done;"
	if initContainer[0].Command[2] != expectedCMD {
		t.Errorf("Expected %s , got %s", expectedCMD, initContainer[0].Command[2])
	}
}
