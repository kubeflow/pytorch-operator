package config

import (
	"io/ioutil"

	log "github.com/sirupsen/logrus"
)

var initContainerTemplate = `
- name: init-pytorch
  image: busybox:1.31.0
  imagePullPolicy: IfNotPresent
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
