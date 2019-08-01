local params = import '../../components/params.libsonnet';

params {
  components+: {
    workflows+: {
      namespace: 'kubeflow-test-infra',
      name: 'pytorch-operator-release-6aa39a41-6985-kunming',
      prow_env: 'JOB_NAME=pytorch-operator-release,JOB_TYPE=pytorch-operator-release,REPO_NAME=pytorch-operator,REPO_OWNER=kubeflow,BUILD_NUMBER=6985,PULL_BASE_SHA=6aa39a41',
      versionTag: 'v20190703-6aa39a41',
      registry: 'gcr.io/kubeflow-images-public',
      bucket: 'kubeflow-releasing-artifacts',
    },
    "workflows-v1alpha2"+: {
      registry: 'gcr.io/kubeflow-images-public',
      bucket: 'kubeflow-releasing-artifacts',
    },
  },
}