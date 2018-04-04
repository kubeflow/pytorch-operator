package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gogo/protobuf/proto"
	log "github.com/golang/glog"
	torchv1alpha1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1alpha1"
	torchjobclient "github.com/kubeflow/pytorch-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/pytorch-operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var (
	name      = flag.String("name", "", "The name for the PyTorchJob to create..")
	namespace = flag.String("namespace", "kubeflow", "The namespace to create the test job in.")
	numJobs   = flag.Int("num_jobs", 1, "The number of jobs to run.")
	timeout   = flag.Duration("timeout", 10*time.Minute, "The timeout for the test")
)

type tfReplicaType torchv1alpha1.PyTorchReplicaType

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

	tfJobClient, err := torchjobclient.NewForConfig(config)
	if err != nil {
		return "", err
	}
	// Create PyTorchJob from examples
	cmd := exec.Command(
		"kubectl", "apply",
		"-f",
		"examples/multinode/configmap.yaml",
		"-n",
		*namespace,
	)
	err = runCmd(cmd)
	if err != nil {
		log.Errorf("Creating the configmap failed; %v", err)
		return *name, err
	}
	cmd = exec.Command(
		"kubectl", "create",
		"-f",
		"examples/pytorchjob.yaml",
		"-n",
		*namespace,
	)
	err = runCmd(cmd)
	if err != nil {
		log.Errorf("Creating the job failed; %v", err)
		return *name, err
	}

	// TODO(jose5918) Wait for completed state
	// Wait for operator to reach running state
	var tfJob *torchv1alpha1.PyTorchJob
	for endTime := time.Now().Add(*timeout); time.Now().Before(endTime); {
		tfJob, err = tfJobClient.KubeflowV1alpha1().PyTorchJobs(*namespace).Get(*name, metav1.GetOptions{})
		if err != nil {
			log.Warningf("There was a problem getting PyTorchJob: %v; error %v", *name, err)
		}

		if tfJob.Status.State == torchv1alpha1.StateSucceeded || tfJob.Status.State == torchv1alpha1.StateFailed {
			log.Infof("job %v finished:\n%v", *name, util.Pformat(tfJob))
			break
		}
		log.Infof("Waiting for job %v to finish:\n%v", *name, util.Pformat(tfJob))
		time.Sleep(5 * time.Second)
	}

	if tfJob == nil {
		return *name, fmt.Errorf("Failed to get PyTorchJob %v", *name)
	}

	if tfJob.Status.State != torchv1alpha1.StateSucceeded {
		// TODO(jlewi): Should we clean up the job.
		return *name, fmt.Errorf("PyTorchJob %v did not succeed;\n %v", *name, util.Pformat(tfJob))
	}

	if tfJob.Spec.RuntimeId == "" {
		return *name, fmt.Errorf("PyTorchJob %v doesn't have a RuntimeId", *name)
	}

	return *name, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func runCmd(cmd *exec.Cmd) error {
	var waitStatus syscall.WaitStatus
	err := cmd.Run()
	if err != nil {
		// Did the command fail because of an unsuccessful exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus = exitError.Sys().(syscall.WaitStatus)
			output, _ := cmd.CombinedOutput()
			log.Infof("exitcode %d: %s", waitStatus.ExitStatus(), string(output))
		}
	} else {
		// Command was successful
		_ = cmd.ProcessState.Sys().(syscall.WaitStatus)
	}
	return err
}

func main() {
	flag.Parse()

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
		case <-time.After(endTime.Sub(time.Now())):
			log.Errorf("Timeout waiting for PyTorchJob to finish.")
			fmt.Println("timeout 2")
		}
	}

	if numSucceded+numFailed < *numJobs {
		log.Errorf("Timeout waiting for jobs to finish; only %v of %v PyTorchJobs completed.", numSucceded+numFailed, *numJobs)
	}

	// Generate TAP (https://testanything.org/) output
	fmt.Println("1..1")
	if numSucceded == *numJobs {
		fmt.Println("ok 1 - Successfully ran PyTorchJob")
	} else {
		fmt.Printf("not ok 1 - Running PyTorchJobs failed \n")
		// Exit with non zero exit code for Helm tests.
		os.Exit(1)
	}
}
