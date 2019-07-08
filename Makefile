REGISTRY ?= "registry.ci.ciscolabs.com/cpsg_ccp-istio-operator"
# tag/version of docker image
TAG ?= $(shell git describe --always --abbrev=7 2>/dev/null || echo devbuild)
HELM_CHART_VERSION ?= $(shell helm inspect chart charts/ccp-istio-operator/ | grep version | awk '{print $$2}')
OS := $(shell uname)
# repo for premerge/postmerge helm charts
HELM_REPO := https://repo-write.ci.ciscolabs.com/CPSG_ccp-istio-operator/

all: manager

# Run tests
test: fmt vet
# download and install kubebuilder on the host to get "make test" pass,
# workaround for kubebuilder upstream bug
# https://github.com/kubernetes-sigs/kubebuilder/issues/326#issuecomment-494878466
ifeq ($(OS), Darwin)
	# Mac OS
	rm -rf kubebuilder_2.0.0-alpha.1_darwin_amd64*
	wget -q https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.0.0-alpha.1/kubebuilder_2.0.0-alpha.1_darwin_amd64.tar.gz
	tar -zxf  kubebuilder_2.0.0-alpha.1_darwin_amd64.tar.gz
	rm -rf kubebuilder_2.0.0-alpha.1_darwin_amd64.tar.gz
	# run go test
	export KUBEBUILDER_ASSETS=`pwd`/kubebuilder_2.0.0-alpha.1_darwin_amd64/bin && \
	  go test ./api/... ./controllers/... -coverprofile cover.out
	rm -rf kubebuilder_2.0.0-alpha.1_darwin_amd64*
else
	rm -rf kubebuilder_2.0.0-alpha.1_linux_amd64*
	wget -q https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.0.0-alpha.1/kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz
	tar -zxf  kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz
	rm -rf kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz
	# run go test
	export KUBEBUILDER_ASSETS=`pwd`/kubebuilder_2.0.0-alpha.1_linux_amd64/bin && \
	  go test ./api/... ./controllers/... -coverprofile cover.out
	rm -rf kubebuilder_2.0.0-alpha.1_linux_amd64*
endif

# Build manager binary at bin/manager
build-binary: fmt vet
	go build -o bin/manager main.go
	# to run the binary do:
	  # kubectl apply -f charts/ccp-istio-operator/templates/crd.yaml
	  # ./bin/manager

# Run ccp-istio-operator go binary against the configured Kubernetes cluster in ~/.kube/config
run-binary: fmt vet
	# create ccp-istio-operator CRD
	kubectl apply -f charts/ccp-istio-operator/templates/crd.yaml
	go run main.go
	# deploy istio CR by doing "kubectl apply -f ccp-istio-cr.yaml"

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Pull locally built docker image and deploy ccp-istio-operator on k8s
deploy-k8s:
	# set image.pullPolicy to "Never" to pull locally built docker image into k8s pod
	helm install charts/ccp-istio-operator/ --name ccp-istio-operator \
	  --set image.repo=${REGISTRY}/ccp-istio-operator \
	  --set-string image.tag=${TAG} \
	  --set image.pullPolicy=Never

# Delete ccp-istio-operator on k8s
delete-k8s:
	-helm delete --purge ccp-istio-operator
	-kubectl delete crd istios.operator.ccp.cisco.com

# Build docker image
docker-build: test
	make clean > /dev/null 2>&1
	make clean > /dev/null 2>&1
	# if using minikube for dev, run:
	#  eval $(minikube docker-env)
	docker build . -t ${REGISTRY}/ccp-istio-operator:${TAG}

# Push docker image
docker-push:
	# make sure that "docker login ${REGISTRY}" works,
	# image is pushed by CI to internal registry https://registry.ci.ciscolabs.com/v2/cpsg_ccp-istio-operator/ccp-istio-operator/tags/list
	docker push ${REGISTRY}/ccp-istio-operator:${TAG}

helm-package:
	helm init --client-only
	helm package --version "${HELM_CHART_VERSION}-${TAG}" charts/ccp-istio-operator/

helm-upload:
	test -n "$(BUILD_TYPE)"  # $$BUILD_TYPE
	# CI uploads helm charts to https://repo.ci.ciscolabs.com/CPSG_ccp-istio-operator/
	curl --fail -u $(HELM_REPO_USERNAME):$(HELM_REPO_PASSWORD) -X POST -F "file=@ccp-istio-operator-${HELM_CHART_VERSION}-${TAG}.tgz" -F "path=$(BUILD_TYPE)/" $(HELM_REPO)

# install and test istio CR
test-istio-cr:
	echo "======== Installing istio CR ========"
	./tests/test_istio_cr.sh create
	sleep 30
	echo -e "\n\n======== Deleting istio CR ========\n\n"
	./tests/test_istio_cr.sh delete

# Delete ccp-istio-operator on k8s, docker images and other unwanted files
clean: delete-k8s
	-docker images --format "{{.ID}} {{.Repository}} {{.Tag}}" | \
	  grep '<none>\|ccp-istio-operator\|golang\|debian.*9.9-slim' | \
	  awk '{print $1}' | xargs docker rmi -f
	rm -rf bin kubebuilder_2.0.0-alpha.1_* kustomize cover.out \
	       istio-values.yaml istio-init-values.yaml ccp-istio-operator-*.tgz

######################################################################
# Kubebuilder's code generation make targets.                        #
#                                                                    #
# Run "make" and "make manifests" to generate code and k8s manifests #
# in config directory.                                               #
#                                                                    #
# Istio operator's CRD will be generated at                          #
# config/crd/bases/operator.ccp.cisco.com_istios.yaml                #
#                                                                    #
# Istio operator's CRD charts/ccp-istio-operator/templates/crd.yaml  #
# is a copy of config/crd/bases/operator.ccp.cisco.com_istios.yaml   #
######################################################################

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Generate manifests e.g. CRD, RBAC etc.
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./api/...;./controllers/..." output:crd:artifacts:config=config/crd/bases
	cp config/crd/bases/operator.ccp.cisco.com_istios.yaml charts/ccp-istio-operator/templates/crd.yaml

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
