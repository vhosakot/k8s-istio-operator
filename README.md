# CCP istio-operator

Kubernetes operator to manage [istio service mesh](https://istio.io) in a k8s cluster.

This operator can be used to install, upgrade, repair, reconfigure and uninstall istio service mesh in a kubernetes cluster.

This repo was created using [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder), an SDK framework for building Kubernetes APIs using [custom resource definitions (CRDs)](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions).

[kubebuilder.md](https://wwwin-github.cisco.com/CPSG/ccp-istio-operator/blob/master/kubebuilder.md) has the steps that show how this repo was created using kubebuilder.

## Steps to develop CCP istio-operator

CCP istio-operator runs in a docker container and needs istio helm charts on its host which will be mounted inside the container. Download the istio helm charts at `/opt/ccp/charts/` on the host:

### If using docker desktop on Mac

```
sudo mkdir -p /opt/ccp/charts
sudo wget https://repo.ci.ciscolabs.com/CPSG_ccp-charts/upstream/istio-init-1.1.8-ccp1.tgz
sudo wget https://repo.ci.ciscolabs.com/CPSG_ccp-charts/upstream/istio-1.1.8-ccp1.tgz
sudo mv istio-init-1.1.8-ccp1.tgz /opt/ccp/charts/
sudo mv istio-1.1.8-ccp1.tgz /opt/ccp/charts/

$ ls -l /opt/ccp/charts/istio-*
-rw-r--r--@ 1 root  staff  81308 Jun 21 18:41 /opt/ccp/charts/istio-1.1.8-ccp1.tgz
-rw-r--r--@ 1 root  staff   9541 Jun 21 18:41 /opt/ccp/charts/istio-init-1.1.8-ccp1.tgz
```

In the docker GUI on Mac, add the path `/opt/ccp/charts` in `Preferences --> File Sharing`.

### If using minikube

```
wget https://repo.ci.ciscolabs.com/CPSG_ccp-charts/upstream/istio-init-1.1.8-ccp1.tgz
wget https://repo.ci.ciscolabs.com/CPSG_ccp-charts/upstream/istio-1.1.8-ccp1.tgz
minikube ssh "sudo mkdir -p /opt/ccp/charts/ && sudo chmod 777 /opt/ccp/charts/"
scp -o StrictHostKeyChecking=no -i $(minikube ssh-key) istio-init-1.1.8-ccp1.tgz docker@$(minikube ip):/opt/ccp/charts
scp -o StrictHostKeyChecking=no -i $(minikube ssh-key) istio-1.1.8-ccp1.tgz docker@$(minikube ip):/opt/ccp/charts
rm -rf istio-init-1.1.8-ccp1.tgz istio-1.1.8-ccp1.tgz

$ minikube ssh "ls -l /opt/ccp/charts/"
total 92
-rw-r--r-- 1 docker docker 81308 Jun 29 01:28 istio-1.1.8-ccp1.tgz
-rw-r--r-- 1 docker docker  9541 Jun 29 01:28 istio-init-1.1.8-ccp1.tgz
```

### Install helm and tiller if needed

If helm and tiller are not installed in the k8s cluster, install them.

```
wget https://get.helm.sh/helm-v2.12.2-darwin-amd64.tar.gz
tar -zxvf helm-v2.12.2-darwin-amd64.tar.gz
sudo mv darwin-amd64/helm /usr/local/bin/
rm -rf helm-v2.12.2-darwin-amd64.tar.gz darwin-amd64
helm init

# wait  5 minutes so tiller is installed and running

$ helm version
Client: &version.Version{SemVer:"v2.12.2", GitCommit:"7d2b0c73d734f6586ed222a567c5d103fed435be", GitTreeState:"clean"}
Server: &version.Version{SemVer:"v2.12.2", GitCommit:"7d2b0c73d734f6586ed222a567c5d103fed435be", GitTreeState:"clean"}
```

### Install CCP istio-operator to operate istio on kubernetes

Install [Golang](https://golang.org/dl/) if needed and set `GOPATH` if not set.

```
export GOPATH=`go env GOPATH`

$ echo $GOPATH
/home/ubuntu/go
```

```
mkdir -p $GOPATH/src/wwwin-github.cisco.com/CPSG
cd $GOPATH/src/wwwin-github.cisco.com/CPSG

# add your host's SSH public key in https://wwwin-github.cisco.com/settings/keys

git clone git@wwwin-github.cisco.com:CPSG/ccp-istio-operator.git
cd ccp-istio-operator

# if using minikube, run the following command
eval $(minikube docker-env)
```

Make sure that docker commands like `docker images` can be run without `sudo`. Refer https://docs.docker.com/install/linux/linux-postinstall/#manage-docker-as-a-non-root-user.

```
make docker-build

$ docker images | grep 'TAG\|ccp-istio-operator'
REPOSITORY          TAG        IMAGE ID       CREATED         SIZE
ccp-istio-operator  ab38b67    d52a73a76c35   38 seconds ago  137MB

make deploy-k8s

$ helm ls | grep 'NAME\|ccp-istio-operator'
NAME                REVISION UPDATED                  STATUS  	CHART                     APP VERSION	NAMESPACE
ccp-istio-operator  1        Fri Jun 28 22:19:52 2019 DEPLOYED	ccp-istio-operator-1.0.0  1.0.0      	default

helm status ccp-istio-operator
```

Check `ccp-istio-operator` pod and its CRD.

```
$ kubectl get pods -o wide | grep 'NAME\|ccp-istio-operator'
NAME                                 READY  STATUS   RESTARTS  AGE    IP          NODE      NOMINATED NODE  READINESS GATES
ccp-istio-operator-6cfc7fb957-tztqz  1/1    Running  0         3m35s  172.17.0.8  minikube  <none>          <none>

$ kubectl get crds | grep istios.operator.ccp.cisco.com
istios.operator.ccp.cisco.com       2019-06-29T03:44:04Z
```

Now, CCP istio-operator is installed and can be used to operate (install, upgrade, repair, reconfigure and uninstall) istio on kubernetes.

The istio CR manifests are in the [cr](https://wwwin-github.cisco.com/CPSG/ccp-istio-operator/tree/master/cr) directory and are named according to the istio release.

Install istio `1.1.8` using its CR.

```
kubectl apply -f cr/ccp-istio-1.1.8-cr.yaml 

$ kubectl get istio
NAME        AGE   STATUS                    VERSION
ccp-istio   8s    CleaningIstioPreinstall   istio-1.1.8-ccp1.tgz
```

Check istio pods.

```
$ kubectl get pods -n=istio-system
NAME                                      READY   STATUS      RESTARTS   AGE
grafana-845d9867d8-6hsrx                  1/1     Running     0          2m37s
istio-citadel-859d6bb754-vm8n8            1/1     Running     0          2m37s
istio-egressgateway-7fbc9d84d6-vn7lq      1/1     Running     0          2m37s
istio-galley-5bf49ddcf5-vqxfl             1/1     Running     0          2m38s
istio-ingressgateway-5f488bd674-pbnmd     1/1     Running     0          2m37s
istio-init-crd-10-7g55n                   0/1     Completed   0          3m
istio-init-crd-11-4rw5h                   0/1     Completed   0          3m
istio-pilot-9f4675ff9-lk58t               2/2     Running     0          2m37s
istio-policy-6ff478d96b-9t779             2/2     Running     2          2m37s
istio-sidecar-injector-7d59c5688c-277gz   1/1     Running     0          2m37s
istio-telemetry-86f5d4f456-gpm2m          2/2     Running     2          2m37s
prometheus-5989d5fdb7-w7kqh               1/1     Running     0          2m37s
```

After 3-5 minutes, when all the istio pods are in `Running` state, the istio CR's status will be `IstioInstalledActive`.

```
$ kubectl get istio
NAME        AGE     STATUS                 VERSION
ccp-istio   4m19s   IstioInstalledActive   istio-1.1.8-ccp1.tgz

kubectl get istio -o yaml
```

### Update or tweak istio's configurations using istio CR

If istio's configurations need to be updated or tweaked, update the istio CR `cr/ccp-istio-1.1.8-cr.yaml` as needed and apply it again by doing `kubectl apply -f cr/ccp-istio-1.1.8-cr.yaml`.

```
# disable istio's ingress and egress gateways in the istio CR cr/ccp-istio-1.1.8-cr.yaml in
# the spec.istio.values.gateways section
    values: |-
      gateways:
        istio-egressgateway:
          enabled: false
        enabled: false

# apply the updated istio CR
kubectl apply -f cr/ccp-istio-1.1.8-cr.yaml
```

Wait 3-5 minutes and istio will be re-deployed without ingress and egress gateways (`istio-ingressgateway` and `istio-egressgateway` pods will not be running).

```
kubectl get pods -n=istio-system

$ kubectl get istio
NAME        AGE   STATUS                 VERSION
ccp-istio   34m   IstioInstalledActive   istio-1.1.8-ccp1.tgz
```

Istio's configurations can also be updated or tweaked by doing `kubectl edit istio ccp-istio` and istio will be re-deployed with the new/updated configuration in the istio CR `ccp-istio`.

### Check status of istio CR

When istio is successfully installed, the status of istio CR will be `IstioInstalledActive`.

```
$ kubectl get istio ccp-istio -o=jsonpath={.status}
map[active:IstioInstalledActive observedGeneration:2 version:istio-1.1.8-ccp1.tgz]

$ kubectl get istio ccp-istio -o json
...
    "status": {
        "active": "IstioInstalledActive",
        "observedGeneration": 2,
        "version": "istio-1.1.8-ccp1.tgz"
    }
...
```

### Upgrade istio using istio operator

Below are the steps to upgrade istio from `1.1.3` to `1.1.8` using this istio operator.

Download istio `1.1.3` helm charts at `/opt/ccp/charts/` on the host using the steps at the top of this page. Install istio `1.1.3`.

```
# delete istio 1.1.8 if it exists
kubectl delete -f cr/ccp-istio-1.1.8-cr.yaml 

$ kubectl get istio
No resources found.

# install istio 1.1.3 using its CR
kubectl apply -f cr/ccp-istio-1.1.3-cr.yaml 

# check if istio 1.1.3 images are installed by the istio operator
$ kubectl get istio
NAME        AGE     STATUS                 VERSION
ccp-istio   6m31s   IstioInstalledActive   istio-1.1.3-ccp1.tgz

$ kubectl describe pods -n=istio-system | grep Image: | sort | uniq
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-charts/busybox:1.30.1
    Image:          registry.ci.ciscolabs.com/cpsg_ccp-charts/grafana/grafana:6.0.0
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-charts/prom/prometheus:v2.7.1
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-docker-istio/citadel:1.1.3-ccp1
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-docker-istio/galley:1.1.3-ccp1
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-docker-istio/kubectl:1.1.3-ccp1
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-docker-istio/mixer:1.1.3-ccp1
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-docker-istio/pilot:1.1.3-ccp1
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-docker-istio/proxyv2:1.1.3-ccp1
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-docker-istio/sidecar_injector:1.1.3-ccp1
```

Now, to upgrade istio to `1.1.8`, just apply its CR `cr/ccp-istio-1.1.8-cr.yaml`.

```
kubectl apply -f cr/ccp-istio-1.1.8-cr.yaml 

# check if istio 1.1.8 images are installed by the istio operator
$ kubectl get istio
NAME        AGE   STATUS                 VERSION
ccp-istio   11m   IstioInstalledActive   istio-1.1.8-ccp1.tgz

$ kubectl describe pods -n=istio-system | grep Image: | sort | uniq
    Image:          registry.ci.ciscolabs.com/cpsg_ccp-charts/grafana/grafana:6.0.0
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-charts/prom/prometheus:v2.7.1
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-docker-istio/citadel:1.1.8-ccp1
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-docker-istio/galley:1.1.8-ccp1
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-docker-istio/kubectl:1.1.8-ccp1
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-docker-istio/mixer:1.1.8-ccp1
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-docker-istio/pilot:1.1.8-ccp1
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-docker-istio/proxyv2:1.1.8-ccp1
    Image:         registry.ci.ciscolabs.com/cpsg_ccp-docker-istio/sidecar_injector:1.1.8-ccp1
```

Istio has been upgraded from `1.1.3` to `1.1.8`!

### Delete istio, istio CR, CCP istio-operator and cleanup

Delete istio CR.

```
$ kubectl get istio
NAME        AGE   STATUS                 VERSION
ccp-istio   16m   IstioInstalledActive   istio-1.1.8-ccp1.tgz

# delete istio CR
$ kubectl delete -f cr/ccp-istio-1.1.8-cr.yaml 

$ kubectl get istio
No resources found.

# check if istio is deleted
$ kubectl get pods -n=istio-system
No resources found.

$ kubectl get all -n=istio-system
No resources found.
```

Delete CCP istio-operator.

```
make delete-k8s 

$ helm ls | grep ccp-istio-operator
$

$ kubectl get pods --all-namespaces | grep ccp-istio-operator
$
```

Delete docker image.

```
make clean
```

### Running CCP istio-operator as a binary on the host outside the container/k8s pod

```
run-binary

## OR ##

make build-binary
kubectl apply -f charts/ccp-istio-operator/templates/crd.yaml
./bin/manager
```

Running CCP istio-operator as a binary outside the k8s pod is not supported currently as the k8s APIs used by the istio operator talk to the kubernetes api-server, and k8s APIs currently authenticate and work only inside a kubernetes pod (which has the right service account mounted and the environment variables `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT` needed for k8s APIs to work).
