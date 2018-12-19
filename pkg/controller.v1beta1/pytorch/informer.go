package pytorch

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	restclientset "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	v1beta1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1beta1"
	"github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/validation"
	jobinformers "github.com/kubeflow/pytorch-operator/pkg/client/informers/externalversions"
	jobinformersv1beta1 "github.com/kubeflow/pytorch-operator/pkg/client/informers/externalversions/kubeflow/v1beta1"
	"github.com/kubeflow/pytorch-operator/pkg/common/util/unstructured"
	pylogger "github.com/kubeflow/tf-operator/pkg/logger"
)

const (
	resyncPeriod     = 30 * time.Second
	failedMarshalMsg = "Failed to marshal the object to PyTorchJob: %v"
)

var (
	errGetFromKey    = fmt.Errorf("Failed to get PyTorchJob from key")
	errNotExists     = fmt.Errorf("The object is not found")
	errFailedMarshal = fmt.Errorf("Failed to marshal the object to PyTorchJob")
)

func NewUnstructuredPyTorchJobInformer(restConfig *restclientset.Config, namespace string) jobinformersv1beta1.PyTorchJobInformer {
	dynClientPool := dynamic.NewDynamicClientPool(restConfig)
	dclient, err := dynClientPool.ClientForGroupVersionKind(v1beta1.SchemeGroupVersionKind)
	if err != nil {
		panic(err)
	}
	resource := &metav1.APIResource{
		Name:         v1beta1.Plural,
		SingularName: v1beta1.Singular,
		Namespaced:   true,
		Group:        v1beta1.GroupName,
		Version:      v1beta1.GroupVersion,
	}
	informer := unstructured.NewPyTorchJobInformer(
		resource,
		dclient,
		namespace,
		resyncPeriod,
		cache.Indexers{},
	)

	return informer
}

// NewPyTorchJobInformer returns PyTorchJobInformer from the given factory.
func (pc *PyTorchController) NewPyTorchJobInformer(jobInformerFactory jobinformers.SharedInformerFactory) jobinformersv1beta1.PyTorchJobInformer {
	return jobInformerFactory.Kubeflow().V1beta1().PyTorchJobs()
}

func (pc *PyTorchController) getPyTorchJobFromName(namespace, name string) (*v1beta1.PyTorchJob, error) {
	key := fmt.Sprintf("%s/%s", namespace, name)
	return pc.getPyTorchJobFromKey(key)
}

func (pc *PyTorchController) getPyTorchJobFromKey(key string) (*v1beta1.PyTorchJob, error) {
	// Check if the key exists.
	obj, exists, err := pc.jobInformer.GetIndexer().GetByKey(key)
	logger := pylogger.LoggerForKey(key)
	if err != nil {
		logger.Errorf("Failed to get PyTorchJob '%s' from informer index: %+v", key, err)
		return nil, errGetFromKey
	}
	if !exists {
		// This happens after a job was deleted, but the work queue still had an entry for it.
		return nil, errNotExists
	}

	return jobFromUnstructured(obj)
}

func jobFromUnstructured(obj interface{}) (*v1beta1.PyTorchJob, error) {
	// Check if the spec is valid.
	un, ok := obj.(*metav1unstructured.Unstructured)
	if !ok {
		log.Errorf("The object in index is not an unstructured; %+v", obj)
		return nil, errGetFromKey
	}
	var job v1beta1.PyTorchJob
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, &job)
	logger := pylogger.LoggerForUnstructured(un, v1beta1.Kind)
	if err != nil {
		logger.Errorf(failedMarshalMsg, err)
		return nil, errFailedMarshal
	}

	err = validation.ValidateBetaOnePyTorchJobSpec(&job.Spec)
	if err != nil {
		logger.Errorf(failedMarshalMsg, err)
		return nil, errFailedMarshal
	}
	return &job, nil
}

func unstructuredFromPyTorchJob(obj interface{}, job *v1beta1.PyTorchJob) error {
	un, ok := obj.(*metav1unstructured.Unstructured)
	logger := pylogger.LoggerForJob(job)
	if !ok {
		logger.Warn("The object in index isn't type Unstructured")
		return errGetFromKey
	}

	var err error
	un.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(job)
	if err != nil {
		logger.Error("The PyTorchJob convert failed")
		return err
	}
	return nil

}
