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
	"context"
	"fmt"
	"os"
	"time"

	kubebatchclient "github.com/kubernetes-sigs/kube-batch/pkg/client/clientset/versioned"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	kubeinformers "k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclientset "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	election "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"github.com/kubeflow/pytorch-operator/cmd/pytorch-operator.v1/app/options"
	pyv1 "github.com/kubeflow/pytorch-operator/pkg/apis/pytorch/v1"
	jobclientset "github.com/kubeflow/pytorch-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/pytorch-operator/pkg/client/clientset/versioned/scheme"
	jobinformers "github.com/kubeflow/pytorch-operator/pkg/client/informers/externalversions"
	controller "github.com/kubeflow/pytorch-operator/pkg/controller.v1/pytorch"
	"github.com/kubeflow/tf-operator/pkg/util/signals"
	"github.com/kubeflow/tf-operator/pkg/version"
)

const (
	apiVersion = "v1"
)

var (
	// leader election config
	leaseDuration = 15 * time.Second
	renewDuration = 5 * time.Second
	retryPeriod   = 3 * time.Second
	isLeader      = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pytorch_operator_is_leader",
		Help: "Is this client the leader of this pytorch-operator client set?",
	})
)

const RecommendedKubeConfigPathEnv = "KUBECONFIG"

func Run(opt *options.ServerOption) error {
	// Check if the -version flag was passed and, if so, print the version and exit.
	if opt.PrintVersion {
		version.PrintVersionAndExit(apiVersion)
	}

	namespace := os.Getenv(pyv1.EnvKubeflowNamespace)
	if len(namespace) == 0 {
		log.Infof("EnvKubeflowNamespace not set, use default namespace")
		namespace = metav1.NamespaceDefault
	}

	// To help debugging, immediately log version.
	log.Infof("%+v", version.Info(apiVersion))

	// Set up signals so we handle the first shutdown signal gracefully.
	stopCh := signals.SetupSignalHandler()

	// Note: ENV KUBECONFIG will overwrite user defined Kubeconfig option.
	if len(os.Getenv(RecommendedKubeConfigPathEnv)) > 0 {
		// use the current context in kubeconfig
		// This is very useful for running locally.
		opt.Kubeconfig = os.Getenv(RecommendedKubeConfigPathEnv)
	}

	// Get kubernetes config.
	kcfg, err := clientcmd.BuildConfigFromFlags(opt.MasterURL, opt.Kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	// Set client qps and burst by opt.
	kcfg.QPS = float32(opt.QPS)
	kcfg.Burst = opt.Burst

	// Create clients.
	kubeClientSet, leaderElectionClientSet, pytorchJobClientSet, kubeBatchClientSet, err := createClientSets(kcfg)
	if err != nil {
		return err
	}
	if !checkCRDExists(pytorchJobClientSet, opt.Namespace) {
		log.Info("CRD doesn't exist. Exiting")
		os.Exit(1)
	}
	// Create informer factory.
	kubeInformerFactory := kubeinformers.NewFilteredSharedInformerFactory(kubeClientSet, opt.ResyncPeriod, opt.Namespace, nil)
	pytorchJobInformerFactory := jobinformers.NewSharedInformerFactory(pytorchJobClientSet, opt.ResyncPeriod)

	unstructuredInformer := controller.NewUnstructuredPyTorchJobInformer(kcfg, opt.Namespace)

	// Create pytorch controller.
	tc := controller.NewPyTorchController(unstructuredInformer, kubeClientSet, kubeBatchClientSet, pytorchJobClientSet, kubeInformerFactory, pytorchJobInformerFactory, *opt)

	// Start informer goroutines.
	go kubeInformerFactory.Start(stopCh)

	go unstructuredInformer.Informer().Run(stopCh)

	// Set leader election start function.
	run := func(context.Context) {
		isLeader.Set(1)
		if err := tc.Run(opt.Threadiness, stopCh); err != nil {
			log.Errorf("Failed to run the controller: %v", err)
		}
	}

	id, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %v", err)
	}
	// add a uniquifier so that two processes on the same host don't accidentally both become active
	id = id + "_" + string(uuid.NewUUID())

	// Prepare event clients.
	eventBroadcaster := record.NewBroadcaster()
	if err = v1.AddToScheme(scheme.Scheme); err != nil {
		return fmt.Errorf("coreV1 Add Scheme failed: %v", err)
	}
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "pytorch-operator"})

	rl := &resourcelock.EndpointsLock{
		EndpointsMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "pytorch-operator",
		},
		Client: leaderElectionClientSet.CoreV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		},
	}

	// Start leader election.
	election.RunOrDie(context.TODO(), election.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: leaseDuration,
		RenewDeadline: renewDuration,
		RetryPeriod:   retryPeriod,
		Callbacks: election.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				isLeader.Set(0)
				log.Fatalf("leader election lost")
			},
		},
	})

	return nil
}

func createClientSets(config *restclientset.Config) (kubeclientset.Interface, kubeclientset.Interface, jobclientset.Interface, kubebatchclient.Interface, error) {

	kubeClientSet, err := kubeclientset.NewForConfig(restclientset.AddUserAgent(config, "pytorch-operator"))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	leaderElectionClientSet, err := kubeclientset.NewForConfig(restclientset.AddUserAgent(config, "leader-election"))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	jobClientSet, err := jobclientset.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	kubeBatchClientSet, err := kubebatchclient.NewForConfig(restclientset.AddUserAgent(config, "kube-batch"))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return kubeClientSet, leaderElectionClientSet, jobClientSet, kubeBatchClientSet, nil
}

func checkCRDExists(clientset jobclientset.Interface, namespace string) bool {
	_, err := clientset.KubeflowV1().PyTorchJobs(namespace).List(metav1.ListOptions{})

	if err != nil {
		log.Error(err)
		if _, ok := err.(*errors.StatusError); ok {
			if errors.IsNotFound(err) {
				return false
			}
		}
	}
	return true
}
