module github.com/kubeflow/pytorch-operator

go 1.13

require (
	cloud.google.com/go v0.44.0 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20170808103936-bb23615498cd // indirect
	github.com/chai2010/gettext-go v0.0.0-20160711120539-c6fed771bfd5 // indirect
	github.com/codedellemc/goscaleio v0.0.0-20170830184815-20e2ce2cf885 // indirect
	github.com/d2g/dhcp4 v0.0.0-20170904100407-a1d1b6c41b1c // indirect
	github.com/d2g/dhcp4client v0.0.0-20170829104524-6e570ed0a266 // indirect
	github.com/daviddengcn/go-colortext v0.0.0-20160507010035-511bcaf42ccd // indirect
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/fatih/camelcase v0.0.0-20160318181535-f6a740d52f96 // indirect
	github.com/go-openapi/spec v0.19.3
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/golang/protobuf v1.4.2
	github.com/golangplus/bytes v0.0.0-20160111154220-45c989fe5450 // indirect
	github.com/golangplus/fmt v0.0.0-20150411045040-2a5d6d7d2995 // indirect
	github.com/golangplus/testing v0.0.0-20180327235837-af21d9c3145e // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/jteeuwen/go-bindata v0.0.0-20151023091102-a0ff2567cfb7 // indirect
	github.com/kardianos/osext v0.0.0-20150410034420-8fef92e41e22 // indirect
	github.com/kr/fs v0.0.0-20131111012553-2788f0dbd169 // indirect
	github.com/kubeflow/common v0.3.3-0.20210227095238-97658773cce1
	github.com/kubeflow/tf-operator v1.1.0
	github.com/kubernetes-sigs/kube-batch v0.4.2
	github.com/kubernetes-sigs/yaml v1.1.0
	github.com/mailru/easyjson v0.0.0-20190626092158-b2ccc519800e // indirect
	github.com/mholt/caddy v0.0.0-20180213163048-2de495001514 // indirect
	github.com/mitchellh/go-wordwrap v0.0.0-20150314170334-ad45545899c7 // indirect
	github.com/onrik/logrus v0.4.0
	github.com/pkg/sftp v0.0.0-20160930220758-4d0e916071f6 // indirect
	github.com/prometheus/client_golang v1.7.1
	github.com/shurcooL/sanitized_anchor_name v0.0.0-20151028001915-10ef21a441db // indirect
	github.com/sigma/go-inotify v0.0.0-20181102212354-c87b6cf5033d // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/vmware/photon-controller-go-sdk v0.0.0-20170310013346-4a435daef6cc // indirect
	github.com/xanzy/go-cloudstack v0.0.0-20160728180336-1e2cbf647e57 // indirect
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	k8s.io/api v0.19.9
	k8s.io/apimachinery v0.19.9
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6

	k8s.io/utils v0.0.0-20190809000727-6c36bc71fc4a // indirect
)

replace (
github.com/kubeflow/pytorch-operator => ./github.com/paipaoso/pytorch-operator
	k8s.io/api => k8s.io/api v0.16.9
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
	k8s.io/kube-aggregatr => k8s.io/kube-aggregator v0.15.9
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.15.9
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190228160746-b3a7cee44a30
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
