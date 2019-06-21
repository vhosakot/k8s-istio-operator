IMG ?= ccp-istio-operator
TAG ?= $(shell git describe --always --abbrev=7 2>/dev/null || echo devbuild)

all: manager

# Run tests
test: fmt vet
	# download and install kubebuilder on the host to get "make test" pass,
	# workaround for kubebuilder upstream bug
	# https://github.com/kubernetes-sigs/kubebuilder/issues/326#issuecomment-494878466
	rm -rf kubebuilder_2.0.0-alpha.1_linux_amd64*
	wget -q https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.0.0-alpha.1/kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz
	tar -zxf  kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz
	rm -rf kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz
	# run go test
	export KUBEBUILDER_ASSETS=`pwd`/kubebuilder_2.0.0-alpha.1_linux_amd64/bin && \
	  go test ./api/... ./controllers/... -coverprofile cover.out
	rm -rf kubebuilder_2.0.0-alpha.1_linux_amd64*

# Build manager binary at bin/manager
build-binary: fmt vet
	go build -o bin/manager main.go
	# to run the binary do:
	  # kubectl apply -f helm/templates/crd.yaml
	  # ./bin/manager

# Run ccp-istio-operator go binary against the configured Kubernetes cluster in ~/.kube/config
run-binary: fmt vet
	# create ccp-istio-operator CRD
	kubectl apply -f helm/templates/crd.yaml
	go run main.go
	# deploy istio CR by doing "kubectl apply -f ccp-istio-cr.yaml"

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Deploy ccp-istio-operator on k8s
deploy-k8s:
	# set image.pullPolicy to "Never" to pull locally built docker image into k8s pod
	helm install ./helm/ --name ccp-istio-operator \
	  --set image.tag=${TAG} \
	  --set image.pullPolicy=Never

# Delete ccp-istio-operator on k8s
delete-k8s:
	-helm delete --purge ccp-istio-operator
	-kubectl delete crd istios.operator.ccp.cisco.com

# Build docker image
docker-build: test
	make clean
	make clean
	# if using minikube for dev, run:
	#  eval $(minikube docker-env)
	#  minikube ssh "sudo mkdir -p /opt/ccp/charts/"
	docker build . -t ${IMG}:${TAG}

# Push docker image
docker-push:
	docker push ${IMG}:${TAG}

# Delete ccp-istio-operator on k8s, docker images and other unwanted files
clean: delete-k8s
	-docker images --format "{{.ID}} {{.Repository}} {{.Tag}}" | \
	  grep '<none>\|ccp-istio-operator\|golang\|debian.*9.9-slim' | \
	  awk '{print $1}' | xargs docker rmi -f
	rm -rf bin kubebuilder_2.0.0-alpha.1_linux_amd64* kustomize cover.out \
	       istio-values.yaml istio-init-values.yaml

######################################################################
# Kubebuilder's code generation make targets.                        #
#                                                                    #
# Run "make" and "make manifests" to generate code and k8s manifests #
# in config directory.                                               #
#                                                                    #
# Istio operator's CRD will be generated at                          #
# config/crd/bases/operator.ccp.cisco.com_istios.yaml                #
#                                                                    #
# Istio operator's CRD helm/templates/crd.yaml is a copy of          #
# config/crd/bases/operator.ccp.cisco.com_istios.yaml                #
######################################################################

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Generate manifests e.g. CRD, RBAC etc.
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./api/...;./controllers/..." output:crd:artifacts:config=config/crd/bases
	cp config/crd/bases/operator.ccp.cisco.com_istios.yaml helm/templates/crd.yaml

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./api/...

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen
CONTROLLER_GEN=$(shell go env GOPATH)/bin/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
