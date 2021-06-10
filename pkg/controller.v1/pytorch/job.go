package pytorch

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	common "github.com/kubeflow/common/pkg/apis/common/v1"
	pyv1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1"
	pylogger "github.com/kubeflow/tf-operator/pkg/logger"
	"github.com/kubeflow/tf-operator/pkg/util/k8sutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	failedMarshalPyTorchJobReason = "InvalidPyTorchJobSpec"
)

var (
	pytorchJobsCreatedCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pytorch_operator_jobs_created_total",
		Help: "Counts number of PyTorch jobs created",
	})
)

// When a pod is added, set the defaults and enqueue the current pytorchjob.
func (pc *PyTorchController) addPyTorchJob(obj interface{}) {
	// Convert from unstructured object.
	job, err := jobFromUnstructured(obj)
	if err != nil {
		un, ok := obj.(*metav1unstructured.Unstructured)
		logger := &log.Entry{}
		if ok {
			logger = pylogger.LoggerForUnstructured(un, pyv1.Kind)
		}
		logger.Errorf("Failed to convert the PyTorchJob: %v", err)
		// Log the failure to conditions.
		if err == errFailedMarshal {
			errMsg := fmt.Sprintf("Failed to unmarshal the object to PyTorchJob: Spec is invalid %v", err)
			logger.Warn(errMsg)
			pc.Recorder.Event(un, v1.EventTypeWarning, failedMarshalPyTorchJobReason, errMsg)

			status := common.JobStatus{
				Conditions: []common.JobCondition{
					{
						Type:               common.JobFailed,
						Status:             v1.ConditionTrue,
						LastUpdateTime:     metav1.Now(),
						LastTransitionTime: metav1.Now(),
						Reason:             failedMarshalPyTorchJobReason,
						Message:            errMsg,
					},
				},
			}

			statusMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&status)

			if err != nil {
				logger.Errorf("Could not covert the PyTorchJobStatus to unstructured; %v", err)
				return
			}

			client, err := k8sutil.NewCRDRestClient(&pyv1.SchemeGroupVersion)

			if err == nil {
				if err1 := metav1unstructured.SetNestedField(un.Object, statusMap, "status"); err1 != nil {
					logger.Errorf("Could not set nested field: %v", err1)
				}
				logger.Infof("Updating the job to: %+v", un.Object)
				err = client.UpdateStatus(un, pyv1.Plural)
				if err != nil {
					logger.Errorf("Could not update the PyTorchJob: %v", err)
				}
			} else {
				logger.Errorf("Could not create a REST client to update the PyTorchJob")
			}
		}
		return
	}

	// Set default for the new job.
	scheme.Scheme.Default(job)

	msg := fmt.Sprintf("PyTorchJob %s is created.", job.Name)
	logger := pylogger.LoggerForJob(job)
	logger.Info(msg)

	// Add a created condition.
	err = updatePyTorchJobConditions(job, common.JobCreated, pytorchJobCreatedReason, msg)
	if err != nil {
		logger.Errorf("Append job condition error: %v", err)
		return
	}

	// Convert from pytorchjob object
	err = unstructuredFromPyTorchJob(obj, job)
	if err != nil {
		logger.Errorf("Failed to convert the obj: %v", err)
		return
	}
	pc.enqueuePyTorchJob(obj)
	pytorchJobsCreatedCount.Inc()
}

// When a pod is updated, enqueue the current pytorchjob.
func (pc *PyTorchController) updatePyTorchJob(old, cur interface{}) {
	oldPyTorchJob, err := jobFromUnstructured(old)
	if err != nil {
		return
	}
	curPyTorchJob, err := jobFromUnstructured(cur)
	if err != nil {
		return
	}

	// never return error
	key, err := KeyFunc(curPyTorchJob)
	if err != nil {
		return
	}

	log.Infof("Updating pytorchjob: %s", oldPyTorchJob.Name)
	pc.enqueuePyTorchJob(cur)

	// check if need to add a new rsync for ActiveDeadlineSeconds
	if curPyTorchJob.Status.StartTime != nil {
		curPyTorchJobADS := curPyTorchJob.Spec.ActiveDeadlineSeconds
		if curPyTorchJobADS == nil {
			return
		}
		oldPyTorchJobADS := oldPyTorchJob.Spec.ActiveDeadlineSeconds
		if oldPyTorchJobADS == nil || *oldPyTorchJobADS != *curPyTorchJobADS {
			now := metav1.Now()
			start := curPyTorchJob.Status.StartTime.Time
			passed := now.Time.Sub(start)
			total := time.Duration(*curPyTorchJobADS) * time.Second
			// AddAfter will handle total < passed
			pc.WorkQueue.AddAfter(key, total-passed)
			log.Infof("job ActiveDeadlineSeconds updated, will rsync after %d seconds", total-passed)
		}
	}
}

// deletePodsAndServices deletes all the pods and master service.
func (pc *PyTorchController) deletePodsAndServices(job *pyv1.PyTorchJob, pods []*v1.Pod, services []*v1.Service) error {
	if len(pods) == 0 {
		return nil
	}

	// Delete nothing when the cleanPodPolicy is None.
	if *job.Spec.CleanPodPolicy == common.CleanPodPolicyNone {
		return nil
	}

	for _, pod := range pods {
		// Just delete running pod when the cleanPodPolicy is Running
		if *job.Spec.CleanPodPolicy == common.CleanPodPolicyRunning && pod.Status.Phase != v1.PodRunning {
			continue
		}
		if err := pc.PodControl.DeletePod(pod.Namespace, pod.Name, job); err != nil {
			return err
		}
	}

	rt := strings.ToLower(string(pyv1.PyTorchReplicaTypeMaster))
	services, err := pc.FilterServicesForReplicaType(services, rt)
	if err != nil {
		return err
	}
	for _, service := range services {
		if err := pc.ServiceControl.DeleteService(service.Namespace, service.Name, job); err != nil {
			return err
		}
	}
	return nil
}

func (pc *PyTorchController) cleanupPyTorchJob(job *pyv1.PyTorchJob) error {
	currentTime := time.Now()
	ttl := job.Spec.TTLSecondsAfterFinished
	if ttl == nil {
		// do nothing if the cleanup delay is not set
		return nil
	}
	duration := time.Second * time.Duration(*ttl)
	if currentTime.After(job.Status.CompletionTime.Add(duration)) {
		err := pc.deletePyTorchJobHandler(job)
		if err != nil {
			pylogger.LoggerForJob(job).Warnf("Cleanup PyTorchJob error: %v.", err)
			return err
		}
		return nil
	}
	key, err := KeyFunc(job)
	if err != nil {
		pylogger.LoggerForJob(job).Warnf("Couldn't get key for pytorchjob object: %v", err)
		return err
	}
	pc.WorkQueue.AddRateLimited(key)
	return nil
}

// deletePyTorchJob deletes the given PyTorchJob.
func (pc *PyTorchController) deletePyTorchJob(job *pyv1.PyTorchJob) error {
	return pc.jobClientSet.KubeflowV1().PyTorchJobs(job.Namespace).Delete(job.Name, &metav1.DeleteOptions{})
}

func getTotalReplicas(job *pyv1.PyTorchJob) int32 {
	jobReplicas := int32(0)
	for _, r := range job.Spec.PyTorchReplicaSpecs {
		jobReplicas += *r.Replicas
	}
	return jobReplicas
}

func getTotalFailedReplicas(job *pyv1.PyTorchJob) int32 {
	totalFailedReplicas := int32(0)
	for rtype := range job.Status.ReplicaStatuses {
		totalFailedReplicas += job.Status.ReplicaStatuses[rtype].Failed
	}
	return totalFailedReplicas
}
