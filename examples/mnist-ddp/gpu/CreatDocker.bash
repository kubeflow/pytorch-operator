echo Please enter the system password first, and then Docker password
sudo docker login -u deepermind
TRAIN_IMAGE=docker.io/deepermind/pytorch-mnist-ddp-gpu
sudo docker build . -f Dockerfile -t ${TRAIN_IMAGE}
sudo docker push ${TRAIN_IMAGE}

