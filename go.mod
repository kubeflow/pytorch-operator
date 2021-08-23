module github.com/kubeflow/pytorch-operator

go 1.13

require (
	cloud.google.com/go v0.44.0 // indirect
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/go-openapi/spec v0.20.3
	github.com/gogo/protobuf v1.3.1
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/kubeflow/common v0.4.0
	github.com/kubeflow/tf-operator v1.2.0
	github.com/kubernetes-sigs/kube-batch v0.4.2
	github.com/kubernetes-sigs/yaml v1.1.0
	github.com/onrik/logrus v0.4.0
	github.com/prometheus/client_golang v1.10.0
	github.com/sirupsen/logrus v1.6.0
	k8s.io/api v0.19.9
	k8s.io/apimachinery v0.19.9
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
)

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.0
	github.com/kubeflow/common => github.com/kubeflow/common v0.4.0
	github.com/kubeflow/pytorch-operator => ../pytorch-operator
	k8s.io/api => k8s.io/api v0.16.9
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.15.10-beta.0
	k8s.io/apiserver => k8s.io/apiserver v0.15.9
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.15.9
	k8s.io/client-go => k8s.io/client-go v0.16.9
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.15.9
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.15.9
	k8s.io/code-generator => k8s.io/code-generator v0.15.13-beta.0
	k8s.io/component-base => k8s.io/component-base v0.15.9
	k8s.io/cri-api => k8s.io/cri-api v0.15.13-beta.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.15.9
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.15.9
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.15.9
	k8s.io/kube-openapi => github.com/paipaoso/kube-openapi v0.1.0
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.15.9
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.15.9
	k8s.io/kubectl => k8s.io/kubectl v0.15.13-beta.0
	k8s.io/kubelet => k8s.io/kubelet v0.15.9
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.15.9
	k8s.io/metrics => k8s.io/metrics v0.15.9
	k8s.io/node-api => k8s.io/node-api v0.15.9
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.15.9
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.15.9
	k8s.io/sample-controller => k8s.io/sample-controller v0.15.9
)
