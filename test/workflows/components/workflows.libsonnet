{
  // TODO(https://github.com/ksonnet/ksonnet/issues/222): Taking namespace as an argument is a work around for the fact that ksonnet
  // doesn't support automatically piping in the namespace from the environment to prototypes.

  // convert a list of two items into a map representing an environment variable
  // TODO(jlewi): Should we move this into kubeflow/core/util.libsonnet
  listToMap:: function(v)
    {
      name: v[0],
      value: v[1],
    },

  // Function to turn comma separated list of prow environment variables into a dictionary.
  parseEnv:: function(v)
    local pieces = std.split(v, ",");
    if v != "" && std.length(pieces) > 0 then
      std.map(
        function(i) $.listToMap(std.split(i, "=")),
        std.split(v, ",")
      )
    else [],


  // default parameters.
  defaultParams:: {
    project:: "kubeflow-ci",
    zone:: "us-east1-d",
    // Default registry to use.
    registry:: "gcr.io/" + $.defaultParams.project,

    // The image tag to use.
    // Defaults to a value based on the name.
    versionTag:: null,

    // The name of the secret containing GCP credentials.
    gcpCredentialsSecretName:: "kubeflow-testing-credentials",
  },

  // overrides is a dictionary of parameters to provide in addition to defaults.
  parts(namespace, name, overrides={}):: {
    // Workflow to run the e2e test.
    e2e(prow_env, bucket):
      local params = $.defaultParams + overrides;
      // mountPath is the directory where the volume to store the test data
      // should be mounted.
      local mountPath = "/mnt/" + "test-data-volume";
      // testDir is the root directory for all data for a particular test run.
      local testDir = mountPath + "/" + name;
      // outputDir is the directory to sync to GCS to contain the output for this job.
      local outputDir = testDir + "/output";
      local artifactsDir = outputDir + "/artifacts";
      local goDir = testDir + "/go";
      // Source directory where all repos should be checked out
      local srcRootDir = testDir + "/src";
      // The directory containing the kubeflow/pytorch-operator repo
      local srcDir = srcRootDir + "/kubeflow/pytorch-operator";
      local testWorkerImage = "gcr.io/kubeflow-ci/test-worker";
      local golangImage = "golang:1.9.4-stretch";
      // TODO(jose5918) Build our own helm image
      local helmImage = "volumecontroller/golang:1.9.2";
      // The name of the NFS volume claim to use for test files.
      // local nfsVolumeClaim = "kubeflow-testing";
      local nfsVolumeClaim = "nfs-external";
      // The name to use for the volume to use to contain test data.
      local dataVolume = "kubeflow-test-volume";
      local versionTag = if params.versionTag != null then
        params.versionTag
      else name;
      local pytorchJobImage = params.registry + "/pytorch_operator:" + versionTag;

      // The namespace on the cluster we spin up to deploy into.
      local deployNamespace = "kubeflow";
      // The directory within the kubeflow_testing submodule containing
      // py scripts to use.
      local k8sPy = srcDir;
      local kubeflowPy = srcRootDir + "/kubeflow/testing/py";

      local project = params.project;
      // GKE cluster to use
      // We need to truncate the cluster to no more than 40 characters because
      // cluster names can be a max of 40 characters.
      // We expect the suffix of the cluster name to be unique salt.
      // We prepend a z because cluster name must start with an alphanumeric character
      // and if we cut the prefix we might end up starting with "-" or other invalid
      // character for first character.
      local cluster =
        if std.length(name) > 40 then
          "z" + std.substr(name, std.length(name) - 39, 39)
        else
          name;
      local zone = params.zone;
      local registry = params.registry;
      local chart = srcDir + "/pytorch-operator-chart";
      {
        // Build an Argo template to execute a particular command.
        // step_name: Name for the template
        // command: List to pass as the container command.
        buildTemplate(step_name, image, command):: {
          name: step_name,
          container: {
            command: command,
            image: image,
            workingDir: srcDir,
            env: [
              {
                // Add the source directories to the python path.
                name: "PYTHONPATH",
                value: k8sPy + ":" + kubeflowPy,
              },
              {
                // Set the GOPATH
                name: "GOPATH",
                value: goDir,
              },
              {
                name: "CLUSTER_NAME",
                value: cluster,
              },
              {
                name: "GCP_ZONE",
                value: zone,
              },
              {
                name: "GCP_PROJECT",
                value: project,
              },
              {
                name: "GCP_REGISTRY",
                value: registry,
              },
              {
                name: "DEPLOY_NAMESPACE",
                value: deployNamespace,
              },
              {
                name: "GOOGLE_APPLICATION_CREDENTIALS",
                value: "/secret/gcp-credentials/key.json",
              },
              {
                name: "GIT_TOKEN",
                valueFrom: {
                  secretKeyRef: {
                    name: "github-token",
                    key: "github_token",
                  },
                },
              },
            ] + prow_env,
            volumeMounts: [
              {
                name: dataVolume,
                mountPath: mountPath,
              },
              {
                name: "github-token",
                mountPath: "/secret/github-token",
              },
              {
                name: "gcp-credentials",
                mountPath: "/secret/gcp-credentials",
              },
            ],
          },
        },  // buildTemplate

        apiVersion: "argoproj.io/v1alpha1",
        kind: "Workflow",
        metadata: {
          name: name,
          namespace: namespace,
        },
        // TODO(jlewi): Use OnExit to run cleanup steps.
        spec: {
          entrypoint: "e2e",
          volumes: [
            {
              name: "github-token",
              secret: {
                secretName: "github-token",
              },
            },
            {
              name: "gcp-credentials",
              secret: {
                secretName: params.gcpCredentialsSecretName,
              },
            },
            {
              name: dataVolume,
              persistentVolumeClaim: {
                claimName: nfsVolumeClaim,
              },
            },
          ],  // volumes
          // onExit specifies the template that should always run when the workflow completes.
          onExit: "exit-handler",
          templates: [
            {
              name: "e2e",
              steps: [
                [{
                  name: "checkout",
                  template: "checkout",
                }],
                [
                  {
                    name: "build",
                    template: "build",
                  },
                  {
                    name: "create-pr-symlink",
                    template: "create-pr-symlink",
                  },
                ],
                [  // Setup cluster needs to run after build because we depend on the chart
                  // created by the build statement.
                  {
                    name: "setup-cluster",
                    template: "setup-cluster",
                  },
                ],
                [
                  {
                    name: "setup-kubeflow",
                    template: "setup-kubeflow",
                  },
                ],
                [
                  {
                    name: "run-v1alpha1-defaults",
                    template: "run-v1alpha1-defaults",
                  },
                ],
                [
                  {
                    name: "setup-v1alpha2",
                    template: "setup-v1alpha2",
                  },
                ],
                [
                  {
                    name: "run-v1alpha2-defaults",
                    template: "run-v1alpha2-defaults",
                  },
                  {
                    name: "run-v1alpha2-cleanpodpolicy-all",
                    template: "run-v1alpha2-cleanpodpolicy-all",
                  },
                ],
              ],
            },
            {
              name: "exit-handler",
              steps: [
                [{
                  name: "teardown-cluster",
                  template: "teardown-cluster",

                }],
                [{
                  name: "copy-artifacts",
                  template: "copy-artifacts",
                }],
              ],
            },
            {
              name: "checkout",
              container: {
                command: [
                  "/usr/local/bin/checkout.sh",
                  srcRootDir,
                ],
                env: prow_env + [{
                  name: "EXTRA_REPOS",
                  value: "kubeflow/testing@HEAD",
                }],
                image: testWorkerImage,
                volumeMounts: [
                  {
                    name: dataVolume,
                    mountPath: mountPath,
                  },
                ],
              },
            },  // checkout
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("setup-cluster", testWorkerImage, [
              "scripts/create-cluster.sh",
            ]),  // setup cluster
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("setup-kubeflow", testWorkerImage, [
              "scripts/setup-kubeflow.sh",
            ]),  // setup kubeflow
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("run-v1alpha1-defaults", testWorkerImage, [
              "scripts/v1alpha1/run-defaults.sh",
            ]),  // run v1alpha1 default tests
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("setup-v1alpha2", testWorkerImage, [
              "scripts/v1alpha2/setup-v1alpha2.sh",
            ]),  // setup operator v1alpha2 version
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("run-v1alpha2-defaults", testWorkerImage, [
              "scripts/v1alpha2/run-defaults.sh",
            ]),  // run v1alpha2 default tests
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("run-v1alpha2-cleanpodpolicy-all", testWorkerImage, [
              "scripts/v1alpha2/run-cleanpodpolicy-all.sh",
            ]),  // run v1alpha2 cleanpodpolicy tests
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("create-pr-symlink", testWorkerImage, [
              "python",
              "-m",
              "kubeflow.testing.prow_artifacts",
              "--artifacts_dir=" + outputDir,
              "create_pr_symlink",
              "--bucket=" + bucket,
            ]),  // create-pr-symlink
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("teardown-cluster", testWorkerImage, [
              "scripts/delete-cluster.sh",
            ]),  // teardown cluster
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("copy-artifacts", testWorkerImage, [
              "python",
              "-m",
              "kubeflow.testing.prow_artifacts",
              "--artifacts_dir=" + outputDir,
              "copy_artifacts",
              "--bucket=" + bucket,
            ]),  // copy-artifacts
            $.parts(namespace, name).e2e(prow_env, bucket).buildTemplate("build", testWorkerImage, [
              "scripts/build.sh",
            ]),  // build
          ],  // templates
        },
      },  // e2e
  },  // parts
}
