package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	pyv1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1"
	torchjobclient "github.com/kubeflow/pytorch-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/pytorch-operator/pkg/util"
	common "github.com/kubeflow/common/job_controller/api/v1"
	"github.com/kubeflow/tf-operator/pkg/common/jobcontroller"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var (
	name      = flag.String("name", "", "The name for the PyTorchJob to create..")
	namespace = flag.String("namespace", "kubeflow", "The namespace to create the test job in.")
	numJobs   = flag.Int("num_jobs", 1, "The number of jobs to run.")
	timeout   = flag.Duration("timeout", 10*time.Minute, "The timeout for the test")
	image     = flag.String("image", "", "The Test image to run")
)

func getReplicaSpec(worker int32) map[pyv1.PyTorchReplicaType]*common.ReplicaSpec {
	spec := make(map[pyv1.PyTorchReplicaType]*common.ReplicaSpec)
	spec[pyv1.PyTorchReplicaTypeMaster] = replicaSpec(1)
	spec[pyv1.PyTorchReplicaTypeWorker] = replicaSpec(worker)
	return spec

}

func replicaSpec(replica int32) *common.ReplicaSpec {
	return &common.ReplicaSpec{
		Replicas: proto.Int32(replica),
		Template: v1.PodTemplateSpec{
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:            "pytorch",
						Image:           *image,
						ImagePullPolicy: "IfNotPresent",
					},
				},
			},
		},
	}

}

func hasCondition(status common.JobStatus, condType common.JobConditionType) bool {
	for _, condition := range status.Conditions {
		if condition.Type == condType && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

func isSucceeded(status common.JobStatus) bool {
	return hasCondition(status, common.JobSucceeded)
}

func isFailed(status common.JobStatus) bool {
	return hasCondition(status, common.JobFailed)
}

func run() (string, error) {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	if *name == "" {
		name = proto.String("example-job")
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	if *image == "" {
		log.Fatalf("--image must be provided.")
	}

	// create the clientset
	client := kubernetes.NewForConfigOrDie(config)

	torchJobClient, err := torchjobclient.NewForConfig(config)
	if err != nil {
		return "", err
	}

	original := &pyv1.PyTorchJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: *name,
		},
		Spec: pyv1.PyTorchJobSpec{
			PyTorchReplicaSpecs: getReplicaSpec(3),
		},
	}
	// Create PyTorchJob
	_, err = torchJobClient.KubeflowV1().PyTorchJobs(*namespace).Create(original)
	if err != nil {
		log.Errorf("Creating the job failed; %v", err)
		return *name, err
	}
	log.Infof("Job created: \n%v", util.Pformat(original))
	var torchJob *pyv1.PyTorchJob
	for endTime := time.Now().Add(*timeout); time.Now().Before(endTime); {
		torchJob, err = torchJobClient.KubeflowV1().PyTorchJobs(*namespace).Get(*name, metav1.GetOptions{})
		if err != nil {
			log.Errorf("There was a problem getting PyTorchJob: %v; error %v", *name, err)
			return *name, err
		}

		if isSucceeded(torchJob.Status) || isFailed(torchJob.Status) {
			log.Infof("job %v finished:\n%v", *name, util.Pformat(torchJob))
			break
		}
		log.Infof("Waiting for job %v to finish", *name)
		time.Sleep(5 * time.Second)
	}

	if torchJob == nil {
		return *name, fmt.Errorf("Failed to get PyTorchJob %v", *name)
	}

	if !isSucceeded(torchJob.Status) {
		return *name, fmt.Errorf("PyTorchJob %v did not succeed;\n %v", *name, util.Pformat(torchJob))
	}

	// Loop over each replica and make sure the expected resources were created.
	for rtype, r := range original.Spec.PyTorchReplicaSpecs {

		for i := 0; i < int(*r.Replicas); i++ {
			podName := jobcontroller.GenGeneralName(torchJob.Name, strings.ToLower(string(rtype)), strconv.Itoa(i))

			_, err := client.CoreV1().Pods(*namespace).Get(podName, metav1.GetOptions{})
			if err != nil {
				return *name, fmt.Errorf("PyTorchJob %v did not create Job %v for ReplicaType %v Index %v", *name, podName, rtype, i)
			}
		}
	}

	// Delete the job and make sure all subresources are properly garbage collected.
	if err := torchJobClient.KubeflowV1().PyTorchJobs(*namespace).Delete(*name, &metav1.DeleteOptions{}); err != nil {
		log.Fatalf("Failed to delete PyTorchJob %v; error %v", *name, err)
	}

	deleted := false
	for endTime := time.Now().Add(*timeout); time.Now().Before(endTime); {
		_, err = torchJobClient.KubeflowV1().PyTorchJobs(*namespace).Get(*name, metav1.GetOptions{})
		if k8s_errors.IsNotFound(err) {
			deleted = true
			break
		} else {
			log.Infof("Job %v still exists", *name)
		}
		time.Sleep(5 * time.Second)
	}

	if !deleted {
		return *name, fmt.Errorf("Deletion of PyTorchJob %v failed", *name)
	}
	return *name, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func main() {

	type Result struct {
		Error error
		Name  string
	}
	c := make(chan Result)

	for i := 0; i < *numJobs; i++ {
		go func() {
			name, err := run()
			if err != nil {
				log.Errorf("Job %v didn't run successfully; %v", name, err)
			} else {
				log.Infof("Job %v ran successfully", name)
			}
			c <- Result{
				Name:  name,
				Error: err,
			}
		}()
	}

	numSucceded := 0
	numFailed := 0

	for endTime := time.Now().Add(*timeout); numSucceded+numFailed < *numJobs && time.Now().Before(endTime); {
		select {
		case res := <-c:
			if res.Error == nil {
				numSucceded += 1
			} else {
				numFailed += 1
			}
		case <-time.After(time.Until(endTime)):
			log.Errorf("Timeout waiting for PyTorchJob to finish.")
		}
	}

	if numSucceded+numFailed < *numJobs {
		log.Errorf("Timeout waiting for jobs to finish; only %v of %v PyTorchJobs completed.", numSucceded+numFailed, *numJobs)
	}

	if numSucceded == *numJobs {
		fmt.Println("Successfully ran PyTorchJob")
	} else {
		fmt.Printf("Running PyTorchJobs failed \n")
		// Exit with non zero exit code for Helm tests.
		os.Exit(1)
	}
}
