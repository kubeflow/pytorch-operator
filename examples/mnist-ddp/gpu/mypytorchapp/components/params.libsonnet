{
  global: {
    // User-defined global parameters; accessible to all component and environments, Ex:
    // replicas: 4,
  },
  components: {
    // Component-level parameters, defined initially from 'ks prototype use ...'
    // Each object below should correspond to a component in the components/ directory
    "pytorch-operator": {
      cloud: "null",
      deploymentNamespace: "null",
      deploymentScope: "cluster",
      disks: "null",
      name: "pytorch-operator",
      namespace: "null",
      pytorchDefaultImage: "null",
      pytorchJobImage: "gcr.io/kubeflow-images-public/pytorch-operator:84d07df",
      pytorchJobVersion: "v1alpha2",
    },
  },
}
