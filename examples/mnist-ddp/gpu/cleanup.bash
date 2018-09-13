#!/usr/bin/env bash
source variables.bash

cd ${APP_NAME}
pwd

ks delete ${KF_ENV} -c tfserving
kubectl get pods -n ${NAMESPACE}

JOB=tf-${APP_NAME}job
ks delete ${KF_ENV} -c ${JOB} 

ks delete ${KF_ENV} -c kubeflow-core
kubectl get pods -n ${NAMESPACE}

ks env rm ${KF_ENV}
kubectl delete namespace ${NAMESPACE} 
