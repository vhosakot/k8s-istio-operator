## How this repo was created using kubebuilder

Kubebuilder repo: https://github.com/kubernetes-sigs/kubebuilder

### Download and install kubebuilder binary

Download and install a released version of the `kubebuilder` binary from https://github.com/kubernetes-sigs/kubebuilder/releases.

```
wget https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.0.0-alpha.1/kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz
tar -zxvf  kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz
sudo mv kubebuilder_2.0.0-alpha.1_linux_amd64/bin/kubebuilder /usr/local/bin/

# check kubebuilder version
$ kubebuilder version
Version: version.Version{KubeBuilderVersion:"2.0.0-alpha.1", KubernetesVendor:"1.14.1", GitCommit:"a39cc1a586046d50a74455da6c44da734d2fb8fc", BuildDate:"2019-05-17T23:21:23Z", GoOs:"unknown", GoArch:"unknown"}

rm -rf kubebuilder_2.0.0-alpha.1_linux_amd64/ kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz
```

### Initialize new kubebuilder project

```
sudo apt install --reinstall build-essential
mkdir test-repo
cd test-repo

kubebuilder init --domain ccp.cisco.com
```

### Create k8s APIs using CRDs

```
kubebuilder create api --group operator --version v1alpha1 --kind Istio
```

### Install istio operator CRD

```
make install

# check istio operator CRD
$ kubectl get crd | grep 'NAME\|istios.operator.ccp.cisco.com'
NAME                                          CREATED AT
istios.operator.ccp.cisco.com                 2019-05-20T20:06:43Z
```

### Run controller manager locally

```
make run
```

### Create istio CR (an instance of the istio CRD)

Open a new terminal on the same host and do the following steps:

```
kubectl apply -f config/samples/operator_v1alpha1_istio.yaml

# check istio CR
$ kubectl get istio
NAME           AGE
istio-sample   1m

# see contents of the istio CR yaml file
$ cat config/samples/operator_v1alpha1_istio.yaml
apiVersion: operator.ccp.cisco.com/v1alpha1
kind: Istio
metadata:
  name: istio-sample
spec:
  # Add fields here
  foo: bar
```

### Delete istio CR (instance of the istio CRD)

```
kubectl delete -f config/samples/operator_v1alpha1_istio.yaml

# check istio CR
$ kubectl get istio
No resources found.
```

### Delete istio operator CRD

```
kubectl delete crd istios.operator.ccp.cisco.com
```

### Check controller manager's CRD reconciliation logs

You should see the following logs in the terminal in which `make run` above was run:

```
2019-05-20T20:08:52.817Z	DEBUG	controller-runtime.controller	Successfully Reconciled	{"controller": "istio-application", "request": "default/istio-sample"}
2019-05-20T20:10:35.942Z	DEBUG	controller-runtime.controller	Successfully Reconciled	{"controller": "istio-application", "request": "default/istio-sample"}
```

### Stop controller manager

Hit `CTRL+C` (^C) in the terminal in which `make run` above was run to stop the controller manager.
