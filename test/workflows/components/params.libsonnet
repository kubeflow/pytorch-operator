{
  global: {},
  // TODO(jlewi): Having the component name not match the TFJob name is confusing.
  // Job names can't have hyphens in the name. Moving forward we should use hyphens
  // not underscores in component names.
  components: {
    // Component-level parameters, defined initially from 'ks prototype use ...'
    // Each object below should correspond to a component in the components/ directory
    workflows: {
      bucket: "kubeflow-ci_temp",
      name: "some-very-very-very-very-very-long-name-jlewi-pytorch-k8s-presubmit-test-374-6e32",
      namespace: "kubeflow-test-infra",
      prow_env: "JOB_NAME=pytorch-k8s-presubmit-test,JOB_TYPE=presubmit,PULL_NUMBER=374,REPO_NAME=k8s,REPO_OWNER=tensorflow,BUILD_NUMBER=6e32",
      versionTag: "",
      pyTorchJobVersion: "v1alpha2",
    },
    // v1alpha2 components
    simple_tfjob_v1alpha2: {
      name: "simple-001",
      namespace: "kubeflow-test-infra",
      image: "",
    },
  },
}
