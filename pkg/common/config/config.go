package config

import (
	"io/ioutil"

	log "github.com/sirupsen/logrus"
)

var initContainerTemplate = `
- name: init-pytorch
  image: {{.InitContainerImage}}
  imagePullPolicy: IfNotPresent
  resources:
    limits:
      cpu: 100m
      memory: 20Mi
    requests:
      cpu: 50m
      memory: 10Mi
  command: ['sh', '-c', 'until nslookup {{.MasterAddr}}; do echo waiting for master; sleep 2; done;']`

func init() {
	bytes, err := ioutil.ReadFile("/etc/config/initContainer.yaml")
	if err != nil {
		log.Info("Using default init container template")
	} else {
		log.Info("Using init container template from /etc/config/initContainer.yaml")
		initContainerTemplate = string(bytes)
	}
}

func GetInitContainerTemplate() string {
	return initContainerTemplate
}
