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
	"fmt"
	"strings"
	"time"

	kubebatchclient "github.com/kubernetes-sigs/kube-batch/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"

	common "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/pytorch-operator/cmd/pytorch-operator.v1/app/options"
	pyv1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1"
	jobclientset "github.com/kubeflow/pytorch-operator/pkg/client/clientset/versioned"
	jobscheme "github.com/kubeflow/pytorch-operator/pkg/client/clientset/versioned/scheme"
	jobinformers "github.com/kubeflow/pytorch-operator/pkg/client/informers/externalversions"
	jobinformersv1 "github.com/kubeflow/pytorch-operator/pkg/client/informers/externalversions/pytorch/v1"
	joblisters "github.com/kubeflow/pytorch-operator/pkg/client/listers/pytorch/v1"
	"github.com/kubeflow/tf-operator/pkg/common/jobcontroller"
	pylogger "github.com/kubeflow/tf-operator/pkg/logger"
	"github.com/kubeflow/tf-operator/pkg/util/k8sutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	controllerName = "pytorch-operator"

	// labels for pods and servers.
	replicaTypeLabel    = "pytorch-replica-type"
	replicaIndexLabel   = "pytorch-replica-index"
	labelGroupName      = "group-name"
	labelPyTorchJobName = "pytorch-job-name"
)

var (
	// KeyFunc is the short name to DeletionHandlingMetaNamespaceKeyFunc.
	// IndexerInformer uses a delta queue, therefore for deletes we have to use this
	// key function but it should be just fine for non delete events.
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

	pytorchJobsDeletedCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pytorch_operator_jobs_deleted_total",
		Help: "Counts number of PyTorch jobs deleted",
	})
)

// PyTorchController is the type for PyTorchJob Controller, which manages
// the lifecycle of PyTorchJobs.
type PyTorchController struct {
	jobcontroller.JobController

	// jobClientSet is a clientset for CRD PyTorchJob.
	jobClientSet jobclientset.Interface

	// To allow injection of sync functions for testing.
	syncHandler func(string) (bool, error)

	// To allow injection of updateStatus for testing.
	updateStatusHandler func(job *pyv1.PyTorchJob) error

	// To allow injection of deletePyTorchJob for testing.
	deletePyTorchJobHandler func(job *pyv1.PyTorchJob) error

	// jobInformer is a temporary field for unstructured informer support.
	jobInformer cache.SharedIndexInformer

	// Listers for PyTorchJob, Pod and Service
	// jobLister can list/get jobs from the shared informer's store.
	jobLister joblisters.PyTorchJobLister

	// jobInformerSynced returns true if the job store has been synced at least once.
	jobInformerSynced cache.InformerSynced

	initContainerImage string
}

// NewPyTorchController returns a new PyTorchJob controller.
func NewPyTorchController(
	// This variable is for unstructured informer.
	jobInformer jobinformersv1.PyTorchJobInformer,
	kubeClientSet kubeclientset.Interface,
	kubeBatchClientSet kubebatchclient.Interface,
	jobClientSet jobclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	// This field is not used now but we keep it since it will be used
	// after we support CRD validation.
	jobInformerFactory jobinformers.SharedInformerFactory,
	option options.ServerOption) *PyTorchController {

	err := jobscheme.AddToScheme(scheme.Scheme)
	if err != nil {
		log.Fatalf("Failed to add pytorchjob scheme: %v", err)
	}

	log.Info("Creating PyTorchJob controller")
	// Create new PyTorchController.
	pc := &PyTorchController{
		jobClientSet: jobClientSet,
	}

	// Create base controller
	log.Info("Creating Job controller")
	jc := jobcontroller.NewJobController(pc, metav1.Duration{Duration: 15 * time.Second},
		option.EnableGangScheduling, option.GangSchedulerName,
		kubeClientSet, kubeBatchClientSet, kubeInformerFactory, pyv1.Plural)
	pc.initContainerImage = option.InitContainerImage
	pc.JobController = jc
	// Set sync handler.
	pc.syncHandler = pc.syncPyTorchJob
	pc.updateStatusHandler = pc.updatePyTorchJobStatus
	// set delete handler.
	pc.deletePyTorchJobHandler = pc.deletePyTorchJob
	// Set up an event handler for when job resources change.
	jobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    pc.addPyTorchJob,
		UpdateFunc: pc.updatePyTorchJob,
		// This will enter the sync loop and no-op,
		// because the job has been deleted from the store.
		DeleteFunc: pc.enqueuePyTorchJob,
	})

	pc.jobInformer = jobInformer.Informer()
	pc.jobLister = jobInformer.Lister()
	pc.jobInformerSynced = jobInformer.Informer().HasSynced

	// Create pod informer.
	podInformer := kubeInformerFactory.Core().V1().Pods()

	// Set up an event handler for when pod resources change
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    jc.AddPod,
		UpdateFunc: jc.UpdatePod,
		DeleteFunc: jc.DeletePod,
	})

	pc.PodLister = podInformer.Lister()
	pc.PodInformerSynced = podInformer.Informer().HasSynced

	// Create service informer.
	serviceInformer := kubeInformerFactory.Core().V1().Services()

	// Set up an event handler for when service resources change.
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    jc.AddService,
		UpdateFunc: jc.UpdateService,
		DeleteFunc: jc.DeleteService,
	})

	pc.ServiceLister = serviceInformer.Lister()
	pc.ServiceInformerSynced = serviceInformer.Informer().HasSynced

	return pc
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (pc *PyTorchController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer pc.WorkQueue.ShutDown()

	// Start the informer factories to begin populating the informer caches.
	log.Info("Starting PyTorchJob controller")

	// Wait for the caches to be synced before starting workers.
	log.Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(stopCh, pc.jobInformerSynced,
		pc.PodInformerSynced, pc.ServiceInformerSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	log.Infof("Starting %v workers", threadiness)
	// Launch workers to process PyTorchJob resources.
	for i := 0; i < threadiness; i++ {
		go wait.Until(pc.runWorker, time.Second, stopCh)
	}

	log.Info("Started workers")
	<-stopCh
	log.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (pc *PyTorchController) runWorker() {
	for pc.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (pc *PyTorchController) processNextWorkItem() bool {
	obj, quit := pc.WorkQueue.Get()
	if quit {
		return false
	}
	defer pc.WorkQueue.Done(obj)

	var key string
	var ok bool
	if key, ok = obj.(string); !ok {
		// As the item in the workqueue is actually invalid, we call
		// Forget here else we'd go into a loop of attempting to
		// process a work item that is invalid.
		pc.WorkQueue.Forget(obj)
		utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		return true
	}

	logger := pylogger.LoggerForKey(key)

	pytorchJob, err := pc.getPyTorchJobFromKey(key)
	if err != nil {
		if err == errNotExists {
			logger.Infof("PyTorchJob has been deleted: %v", key)
			pytorchJobsDeletedCount.Inc()
			return true
		}

		// Log the failure to conditions.
		logger.Errorf("Failed to get PyTorchJob from key %s: %v", key, err)
		if err == errFailedMarshal {
			errMsg := fmt.Sprintf("Failed to unmarshal the object to PyTorchJob object: %v", err)
			pylogger.LoggerForJob(pytorchJob).Warn(errMsg)
			pc.Recorder.Event(pytorchJob, v1.EventTypeWarning, failedMarshalPyTorchJobReason, errMsg)
		}

		return true
	}

	// Sync PyTorchJob to mapch the actual state to this desired state.
	forget, err := pc.syncHandler(key)
	if err == nil {
		if forget {
			pc.WorkQueue.Forget(key)
		}
		return true
	}

	utilruntime.HandleError(fmt.Errorf("error syncing job: %v", err))
	pc.WorkQueue.AddRateLimited(key)

	return true
}

func (pc *PyTorchController) enqueuePyTorchJob(job interface{}) {
	key, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return
	}

	// TODO: we may need add backoff here
	pc.WorkQueue.Add(key)
}

// syncPyTorchJob syncs the job with the given key if it has had its expectations fulfilled, meaning
// it did not expect to see any more of its pods/services created or deleted.
// This function is not meant to be invoked concurrently with the same key.
func (pc *PyTorchController) syncPyTorchJob(key string) (bool, error) {
	startTime := time.Now()
	logger := pylogger.LoggerForKey(key)
	defer func() {
		logger.Infof("Finished syncing job %q (%v)", key, time.Since(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return false, err
	}
	if len(namespace) == 0 || len(name) == 0 {
		return false, fmt.Errorf("invalid job key %q: either namespace or name is missing", key)
	}

	sharedJob, err := pc.getPyTorchJobFromName(namespace, name)
	if err != nil {
		if err == errNotExists {
			logger.Infof("PyTorchJob has been deleted: %v", key)
			pytorchJobsDeletedCount.Inc()
			// jm.expectations.DeleteExpectations(key)
			return true, nil
		}
		return false, err
	}

	job := sharedJob.DeepCopy()
	jobNeedsSync := pc.satisfiedExpectations(job)

	// Set default for the new job.
	scheme.Scheme.Default(job)

	var reconcilePyTorchJobsErr error
	if jobNeedsSync && job.DeletionTimestamp == nil {
		reconcilePyTorchJobsErr = pc.reconcilePyTorchJobs(job)
	}

	if reconcilePyTorchJobsErr != nil {
		return false, reconcilePyTorchJobsErr
	}

	return true, err
}

// reconcilePyTorchJobs checks and updates replicas for each given PyTorchReplicaSpec.
// It will requeue the job in case of an error while creating/deleting pods/services.
func (pc *PyTorchController) reconcilePyTorchJobs(job *pyv1.PyTorchJob) error {
	jobKey, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for pytorch job object %#v: %v", job, err))
		return err
	}

	logger := pylogger.LoggerForJob(job)
	logger.Infof("Reconcile PyTorchJobs %s", job.Name)

	oldStatus := job.Status.DeepCopy()
	pods, err := pc.GetPodsForJob(job)

	if err != nil {
		logger.Warnf("getPodsForPyTorchJob error %v", err)
		return err
	}

	services, err := pc.GetServicesForJob(job)

	if err != nil {
		logger.Warnf("getServicesForPyTorchJob error %v", err)
		return err
	}

	// If the PyTorchJob is terminated, delete all pods and services.
	if isSucceeded(job.Status) || isFailed(job.Status) {
		if err := pc.deletePodsAndServices(job, pods, services); err != nil {
			return err
		}

		if err := pc.cleanupPyTorchJob(job); err != nil {
			return err
		}

		if pc.Config.EnableGangScheduling {
			if err := pc.DeletePodGroup(job); err != nil {
				return err
			}
		}

		// At this point the pods may have been deleted, so if the job succeeded, we need to manually set the replica status.
		// If any replicas are still Active, set their status to succeeded.
		if isSucceeded(job.Status) {
			for rtype := range job.Status.ReplicaStatuses {
				job.Status.ReplicaStatuses[rtype].Succeeded += job.Status.ReplicaStatuses[rtype].Active
				job.Status.ReplicaStatuses[rtype].Active = 0
			}
		}
		if !apiequality.Semantic.DeepEqual(*oldStatus, job.Status) {
			return pc.updateStatusHandler(job)
		}
		return nil
	}

	// retrieve the previous number of retry
	previousRetry := pc.WorkQueue.NumRequeues(jobKey)

	activePods := k8sutil.FilterActivePods(pods)
	active := int32(len(activePods))
	failed := k8sutil.FilterPodCount(pods, v1.PodFailed)
	totalReplicas := getTotalReplicas(job)
	prevReplicasFailedNum := getTotalFailedReplicas(job)

	var failureMessage string
	jobExceedsLimit := false
	exceedsBackoffLimit := false
	pastBackoffLimit := false

	if job.Spec.BackoffLimit != nil {
		jobHasNewFailure := failed > prevReplicasFailedNum
		// new failures happen when status does not reflect the failures and active
		// is different than parallelism, otherwise the previous controller loop
		// failed updating status so even if we pick up failure it is not a new one
		exceedsBackoffLimit = jobHasNewFailure && (active != totalReplicas) &&
			(int32(previousRetry)+1 > *job.Spec.BackoffLimit)

		pastBackoffLimit, err = pc.pastBackoffLimit(job, pods)
		if err != nil {
			return err
		}
	}

	if exceedsBackoffLimit || pastBackoffLimit {
		// check if the number of pod restart exceeds backoff (for restart OnFailure only)
		// OR if the number of failed jobs increased since the last syncJob
		jobExceedsLimit = true
		failureMessage = fmt.Sprintf("PyTorchJob %s has failed because it has reached the specified backoff limit", job.Name)
	} else if pc.pastActiveDeadline(job) {
		failureMessage = fmt.Sprintf("PyTorchJob %s has failed because it was active longer than specified deadline", job.Name)
		jobExceedsLimit = true
	}

	if jobExceedsLimit {
		if err := pc.deletePodsAndServices(job, pods, services); err != nil {
			return err
		}

		if err := pc.cleanupPyTorchJob(job); err != nil {
			return err
		}

		if pc.Config.EnableGangScheduling {
			if err := pc.DeletePodGroup(job); err != nil {
				return err
			}
		}

		pc.Recorder.Event(job, v1.EventTypeNormal, pytorchJobFailedReason, failureMessage)
		if job.Status.CompletionTime == nil {
			now := metav1.Now()
			job.Status.CompletionTime = &now
		}
		err := updatePyTorchJobConditions(job, common.JobFailed, pytorchJobFailedReason, failureMessage)
		if err != nil {
			logger.Infof("Append pytorchjob condition error: %v", err)
			return err
		}
	} else {
		if pc.Config.EnableGangScheduling {
			minAvailableReplicas := getTotalReplicas(job)
			_, err := pc.SyncPodGroup(job, minAvailableReplicas)
			if err != nil {
				logger.Warnf("Sync PodGroup %v: %v", job.Name, err)
			}
		}

		// Save the current state of the replicas
		replicasStatus := make(map[string]v1.PodPhase)

		// Diff current active pods/services with replicas.
		for rtype, spec := range job.Spec.PyTorchReplicaSpecs {
			err = pc.reconcilePods(job, pods, rtype, spec, replicasStatus)
			if err != nil {
				logger.Warnf("reconcilePods error %v", err)
				return err
			}

			// Service is in need only for Master
			if rtype != pyv1.PyTorchReplicaTypeMaster {
				continue
			}
			err = pc.reconcileServices(job, services, rtype, spec)

			if err != nil {
				logger.Warnf("reconcileServices error %v", err)
				return err
			}
		}
	}

	// No need to update the job if the status hasn't changed since last time.
	if !apiequality.Semantic.DeepEqual(*oldStatus, job.Status) {
		return pc.updateStatusHandler(job)
	}
	return nil
}

// satisfiedExpectations returns true if the required adds/dels for the given job have been observed.
// Add/del counts are established by the controller at sync time, and updated as controllees are observed by the controller
// manager.
func (pc *PyTorchController) satisfiedExpectations(job *pyv1.PyTorchJob) bool {
	satisfied := false
	jobKey, err := KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return false
	}

	for rtype := range job.Spec.PyTorchReplicaSpecs {
		// Check the expectations of the pods.
		expectationPodsKey := jobcontroller.GenExpectationPodsKey(jobKey, string(rtype))
		satisfied = satisfied || pc.Expectations.SatisfiedExpectations(expectationPodsKey)

		// Check the expectations of the services.
		expectationServicesKey := jobcontroller.GenExpectationServicesKey(jobKey, string(rtype))
		satisfied = satisfied || pc.Expectations.SatisfiedExpectations(expectationServicesKey)
	}

	return satisfied
}

// pastBackoffLimit checks if container restartCounts sum exceeds BackoffLimit
// this method applies only to pods with restartPolicy == OnFailure or Always
func (pc *PyTorchController) pastBackoffLimit(job *pyv1.PyTorchJob, pods []*v1.Pod) (bool, error) {
	if job.Spec.BackoffLimit == nil {
		return false, nil
	}
	logger := pylogger.LoggerForJob(job)
	result := int32(0)
	for rtype, spec := range job.Spec.PyTorchReplicaSpecs {
		if spec.RestartPolicy != common.RestartPolicyOnFailure && spec.RestartPolicy != common.RestartPolicyAlways {
			logger.Warnf("The restart policy of replica %v of the job %v is not OnFailure or Always. Not counted in backoff limit.", rtype, job.Name)
			continue
		}
		// Convert PyTorchReplicaType to lower string.
		rt := strings.ToLower(string(rtype))
		pods, err := pc.FilterPodsForReplicaType(pods, rt)
		if err != nil {
			return false, err
		}
		for i := range pods {
			po := pods[i]
			if po.Status.Phase == v1.PodRunning || po.Status.Phase == v1.PodPending {
				for j := range po.Status.InitContainerStatuses {
					stat := po.Status.InitContainerStatuses[j]
					result += stat.RestartCount
				}
				for j := range po.Status.ContainerStatuses {
					stat := po.Status.ContainerStatuses[j]
					result += stat.RestartCount
				}
			}
		}
	}

	if *job.Spec.BackoffLimit == 0 {
		return result > 0, nil
	}
	return result >= *job.Spec.BackoffLimit, nil
}

// pastActiveDeadline checks if job has ActiveDeadlineSeconds field set and if it is exceeded.
func (pc *PyTorchController) pastActiveDeadline(job *pyv1.PyTorchJob) bool {
	if job.Spec.ActiveDeadlineSeconds == nil || job.Status.StartTime == nil {
		return false
	}
	now := metav1.Now()
	start := job.Status.StartTime.Time
	duration := now.Time.Sub(start)
	allowedDuration := time.Duration(*job.Spec.ActiveDeadlineSeconds) * time.Second
	return duration >= allowedDuration
}

func (pc *PyTorchController) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	return pc.getPyTorchJobFromName(namespace, name)
}

func (pc *PyTorchController) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	return pc.jobClientSet.KubeflowV1().PyTorchJobs(namespace).Get(name, metav1.GetOptions{})
}

func (pc *PyTorchController) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return pyv1.SchemeGroupVersionKind
}

func (pc *PyTorchController) GetAPIGroupVersion() schema.GroupVersion {
	return pyv1.SchemeGroupVersion
}

func (pc *PyTorchController) GetGroupNameLabelKey() string {
	return labelGroupName
}

// Deprecated function for backwards compatibility. Has to be removed later
func (pc *PyTorchController) GetJobNameLabelKey() string {
	return labelPyTorchJobName
}

func (pc *PyTorchController) GetGroupNameLabelValue() string {
	return pyv1.GroupName
}

func (pc *PyTorchController) GetReplicaTypeLabelKey() string {
	return replicaTypeLabel
}

func (pc *PyTorchController) GetReplicaIndexLabelKey() string {
	return replicaIndexLabel
}

func (pc *PyTorchController) ControllerName() string {
	return controllerName
}
