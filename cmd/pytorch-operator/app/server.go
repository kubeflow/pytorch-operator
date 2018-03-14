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

package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	election "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"github.com/kubeflow/pytorch-operator/cmd/pytorch-operator/app/options"
	"github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1alpha1"
	torchjobclient "github.com/kubeflow/pytorch-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/pytorch-operator/pkg/client/clientset/versioned/scheme"
	informers "github.com/kubeflow/pytorch-operator/pkg/client/informers/externalversions"
	"github.com/kubeflow/pytorch-operator/pkg/controller"
	"github.com/kubeflow/pytorch-operator/pkg/util"
	"github.com/kubeflow/pytorch-operator/pkg/util/k8sutil"
	"github.com/kubeflow/pytorch-operator/version"
)

var (
	leaseDuration = 15 * time.Second
	renewDuration = 5 * time.Second
	retryPeriod   = 3 * time.Second
)

func Run(opt *options.ServerOption) error {

	// Check if the -version flag was passed and, if so, print the version and exit.
	if opt.PrintVersion {
		version.PrintVersionAndExit()
	}

	namespace := os.Getenv(util.EnvKubeflowNamespace)
	if len(namespace) == 0 {
		log.Infof("EnvKubeflowNamespace not set, use default namespace")
		namespace = metav1.NamespaceDefault
	}

	// To help debugging, immediately log version
	log.Infof("%+v", version.Info())

	config, err := k8sutil.GetClusterConfig()
	if err != nil {
		return err
	}

	kubeClient, leaderElectionClient, pytorchJobClient, apiExtensionsclient, err := createClients(config)
	if err != nil {
		return err
	}

	controllerConfig := readControllerConfig(opt.ControllerConfigFile)

	neverStop := make(chan struct{})
	defer close(neverStop)

	pytorchJobInformerFactory := informers.NewSharedInformerFactory(pytorchJobClient, time.Second*30)
	controller, err := controller.New(kubeClient, apiExtensionsclient, pytorchJobClient, *controllerConfig, pytorchJobInformerFactory)
	if err != nil {
		return err
	}

	go pytorchJobInformerFactory.Start(neverStop)

	run := func(stopCh <-chan struct{}) {
		controller.Run(1, stopCh)
	}

	id, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("Failed to get hostname: %v", err)
	}

	// Prepare event clients.
	eventBroadcaster := record.NewBroadcaster()
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "pytorch-operator"})

	rl := &resourcelock.EndpointsLock{
		EndpointsMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "pytorch-operator",
		},
		Client: leaderElectionClient.CoreV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		},
	}

	election.RunOrDie(election.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: leaseDuration,
		RenewDeadline: renewDuration,
		RetryPeriod:   retryPeriod,
		Callbacks: election.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				log.Fatalf("leader election lost")
			},
		},
	})

	return nil
}

func readControllerConfig(controllerConfigFile string) *v1alpha1.ControllerConfig {
	controllerConfig := &v1alpha1.ControllerConfig{}
	if controllerConfigFile != "" {
		log.Infof("Loading controller config from %v.", controllerConfigFile)
		data, err := ioutil.ReadFile(controllerConfigFile)
		if err != nil {
			log.Fatalf("Could not read file: %v. Error: %v", controllerConfigFile, err)
			return controllerConfig
		}
		err = yaml.Unmarshal(data, controllerConfig)
		if err != nil {
			log.Fatalf("Could not parse controller config; Error: %v\n", err)
		}
		log.Infof("ControllerConfig: %v", util.Pformat(controllerConfig))
	} else {
		log.Info("No controller_config_file provided; using empty config.")
	}
	return controllerConfig
}

func createClients(config *rest.Config) (clientset.Interface, clientset.Interface, torchjobclient.Interface, apiextensionsclient.Interface, error) {
	kubeClient, err := clientset.NewForConfig(rest.AddUserAgent(config, "pytorchjob_operator"))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	leaderElectionClient, err := clientset.NewForConfig(rest.AddUserAgent(config, "leader-election"))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	pytorchJobClient, err := torchjobclient.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	apiExtensionsclient, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return kubeClient, leaderElectionClient, pytorchJobClient, apiExtensionsclient, nil
}
