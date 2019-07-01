## Steps to install CCP istio-operator using helm charts

CCP istio-operator runs in a docker container and needs istio helm charts on its host which will be mounted inside the container. Download the istio helm charts at `/opt/ccp/charts/` on the host:

```
sudo mkdir -p /opt/ccp/charts
sudo wget https://repo.ci.ciscolabs.com/CPSG_ccp-charts/upstream/istio-init-1.1.8-ccp1.tgz
sudo wget https://repo.ci.ciscolabs.com/CPSG_ccp-charts/upstream/istio-1.1.8-ccp1.tgz
sudo mv istio-init-1.1.8-ccp1.tgz /opt/ccp/charts/
sudo mv istio-1.1.8-ccp1.tgz /opt/ccp/charts/

$ ls -l /opt/ccp/charts/istio-*
-rw-r--r-- 1 root root 81308 Jun 21 22:41 /opt/ccp/charts/istio-1.1.8-ccp1.tgz
-rw-r--r-- 1 root root  9541 Jun 21 22:41 /opt/ccp/charts/istio-init-1.1.8-ccp1.tgz
```

```
# add your host's SSH public key in https://wwwin-github.cisco.com/settings/keys

git clone git@wwwin-github.cisco.com:CPSG/ccp-istio-operator.git
cd ccp-istio-operator

helm install charts/ccp-istio-operator/ --name ccp-istio-operator \
    --set image.repo=registry.ci.ciscolabs.com/cpsg_ccp-istio-operator/ccp-istio-operator \
    --set-string image.tag=c9d179b

$ helm ls | grep 'NAME\|ccp-istio-operator'
NAME                	REVISION	UPDATED                 	STATUS  	CHART                     	APP VERSION	NAMESPACE
ccp-istio-operator  	1       	Mon Jul  1 15:49:14 2019	DEPLOYED	ccp-istio-operator-1.0.0  	1.0.0      	default  

helm status ccp-istio-operator
```

Check `ccp-istio-operator` pod and its CRD.

```
$ kubectl get pods -o wide | grep 'NAME\|ccp-istio-operator'
NAME                                  READY   STATUS    RESTARTS   AGE     IP            NODE                              NOMINATED NODE   READINESS GATES
ccp-istio-operator-5684596f9d-mttdp   1/1     Running   0          2m28s   192.168.0.9   vhosakot-istio-masterac5bf816f5   <none>           <none>

$ kubectl get crds | grep istios.operator.ccp.cisco.com
istios.operator.ccp.cisco.com                 2019-07-01T15:49:14Z

```

Now, CCP istio-operator is installed and can be used to operate (install, upgrade, repair, reconfigure and uninstall) istio on kubernetes.

The istio CR manifests are in the [cr](https://wwwin-github.cisco.com/CPSG/ccp-istio-operator/tree/master/cr) directory and are named according to the istio release.

Install istio `1.1.8` using its CR.

```
$ kubectl apply -f cr/ccp-istio-1.1.8-cr.yaml 

$ kubectl get istio
NAME        AGE   STATUS                    VERSION
ccp-istio   8s    CleaningIstioPreinstall   istio-1.1.8-ccp1.tgz

```

Check istio pods.

```
$ kubectl get pods -n=istio-system
NAME                                      READY   STATUS      RESTARTS   AGE
grafana-66c767f5f7-k7474                  1/1     Running     0          104s
istio-citadel-7644dc98d5-lctsq            1/1     Running     0          104s
istio-egressgateway-866789fb87-kx6nf      1/1     Running     0          104s
istio-galley-5f8f59c556-4tjkn             1/1     Running     0          104s
istio-ingressgateway-644c4d8bcf-6fm5p     1/1     Running     0          104s
istio-init-crd-10-6hmp7                   0/1     Completed   0          111s
istio-init-crd-11-wcbq6                   0/1     Completed   0          111s
istio-pilot-547ccc576b-7qrpf              2/2     Running     0          104s
istio-policy-574cf95f79-r4d58             2/2     Running     1          104s
istio-sidecar-injector-665f6b9646-5xvqr   1/1     Running     0          104s
istio-telemetry-7469c999d5-xxklp          2/2     Running     2          104s
prometheus-cdd5f44c8-fvfj2                1/1     Running     0          104s
```

After 3-5 minutes, when all the istio pods are in `Running` state, the istio CR's status will be `IstioInstalledActive`.

```
$ kubectl get istio
NAME        AGE   STATUS                 VERSION
ccp-istio   2m    IstioInstalledActive   istio-1.1.8-ccp1.tgz

kubectl get istio -o yaml

# check if istio 1.1.8 images are installed by the istio operator
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

### Day-2 istio operations using CCP istio-operator

* **Update or tweak istio's configurations using istio CR**
    * Refer https://wwwin-github.cisco.com/CPSG/ccp-istio-operator/blob/master/README.md#update-or-tweak-istios-configurations-using-istio-cr

* **Check status of istio CR**
    * Refer https://wwwin-github.cisco.com/CPSG/ccp-istio-operator/blob/master/README.md#check-status-of-istio-cr

* **Upgrade istio using istio operator**
    * Refer https://wwwin-github.cisco.com/CPSG/ccp-istio-operator/blob/master/README.md#upgrade-istio-using-istio-operator

Delete istio, istio CR, CCP istio-operator

```
$ kubectl get istio
NAME        AGE   STATUS                 VERSION
ccp-istio   9m    IstioInstalledActive   istio-1.1.8-ccp1.tgz

# delete istio CR
kubectl delete -f cr/ccp-istio-1.1.8-cr.yaml 

$ kubectl get istio
No resources found.

# check if istio is deleted
$ kubectl get pods -n=istio-system
No resources found.

$ kubectl get all -n=istio-system
No resources found.

# delete CCP istio-operator helm chart and its CRD
helm delete --purge ccp-istio-operator

$ kubectl get crds | grep istios.operator.ccp.cisco.com
$
```

Configurations of CCP istio-operator's helm charts are in `values.yaml` and can be set using `--set foo=bar` with the `helm install` command.
