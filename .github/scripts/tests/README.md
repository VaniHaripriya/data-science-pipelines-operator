# Setup the local environment

All the following commands must be executed in a single terminal instance.

## Increase inotify Limits
To prevent file monitoring issues in development environments (e.g., IDEs or file sync tools), increase inotify limits:
```bash
sudo sysctl fs.inotify.max_user_instances=2280
sudo sysctl fs.inotify.max_user_watches=1255360
```
## Prerequisites
* Kind https://kind.sigs.k8s.io/

## Create kind cluster
```bash
cat <<EOF | kind create cluster --name=kubeflow --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  image: kindest/node:v1.30.6@sha256:b6d08db72079ba5ae1f4a88a09025c0a904af3b52387643c285442afb05ab994
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    apiServer:
      extraArgs:
        "service-account-issuer": "kubernetes.default.svc"
        "service-account-signing-key-file": "/etc/kubernetes/pki/sa.key"
EOF
```

## kubeconfig
Instead of replacing your kubeconfig, we are going to set to a diff file
```bash
kind get kubeconfig --name kubeflow > /tmp/kubeflow-config
export KUBECONFIG=/tmp/kubeflow-config
```
## docker
In order to by pass the docker limit issue while downloading the images. Let's use your credentials
```bash
docker login -u='...' -p='...' quay.io
```

Upload the secret. The following command will return an error. You need to replace `to` with user `username`
```bash
kubectl create secret generic regcred \
--from-file=.dockerconfigjson=$HOME/.docker/config.json \
--type=kubernetes.io/dockerconfigjson
```

## Test environment variables
Replace the `/path/to` in order to match the `data-science-pipelines-operator` folder
```bash
export GIT_WORKSPACE=/path/to/data-science-pipelines-operator
```
The image registry is required because you are running the test locally.
It will build and upload the image to your repository.

Replace `username` with your quay user
```bash
export REGISTRY_ADDRESS=quay.io/username
```

## Run the test
```bash
sh .github/scripts/tests/tests.sh --kind
```

# Debug a test using GoLand
Let's say you wish to debug the `Should create a Pipeline Run` test.
The first step is right click inside the method content and select the menu
`Run 'TestIntegrationTestSuite'`. It will fail because you need to fill some parameters.
Edit the configuration for `TestIntegrationTestSuite/TestPipelineSuccessfulRun/Should_create_a_Pipeline_Run in github.com/opendatahub-io/data-science-pipelines-operator/tests`
````
-k8sApiServerHost=https://127.0.0.1:39873
-kubeconfig=/tmp/kubeflow-config
-DSPANamespace=test-dspa
-DSPAPath=/path/to/data-science-pipelines-operator/tests/resources/dspa-lite.yaml
````
## How to retrieve the parameters above
* `k8sApiServerHost`: inspect the kubeconfig and retrieve the server URL from there
* `kubeconfig`: the path where you stored the output of `kind get kubeconfig`
* `DSPANamespace`: namespace
* `DSPAPath`: full path for the dspa.yaml

`Should create a Pipeline Run`, `DSPANamespace` and `DSPAPath` depends on the test scenario.

If you wish to keep the resources, add `-skipCleanup=True` in the config above.

## If you wish to rerun the test you need to delete the dspa
```bash
$ kubectl delete datasciencepipelinesapplications test-dspa -n test-dspa
datasciencepipelinesapplication.datasciencepipelinesapplications.opendatahub.io "test-dspa" deleted
```