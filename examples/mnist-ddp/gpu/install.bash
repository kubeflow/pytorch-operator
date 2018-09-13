source variables.bash

kubectl create namespace ${NAMESPACE}

ks init ${APP_NAME}
cd ${APP_NAME}
ks env add ${KF_ENV}
ks env set ${KF_ENV} --namespace ${NAMESPACE}

## Public registry that contains the official kubeflow components
ks registry add kubeflow github.com/kubeflow/kubeflow/tree/master/kubeflow

#Pytorch settings
#Creating pkg
ks pkg install kubeflow/pytorch-job
ks generate pytorch-operator pytorch-operator
#ks param set pytorch-operator pytorchJobVersion v1alpha2
ks apply ${KF_ENV} -c pytorch-operator

echo "Check that the PyTorch crd is installed"
kubectl get crd

#Creating a PyTorch Job
kubectl create -f ../v1alpha2/job_mnist_DDP_GPU.yaml -n $NAMESPACE 

echo "Make sure that the pods are running"
kubectl get pods -n ${NAMESPACE}
