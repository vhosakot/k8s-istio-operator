# Image URL to use all building/pushing image targets
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

# Build manager binary
manager: fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: fmt vet
	go run ./main.go

# Install istio CRD into a cluster
install-istio-crd:
	kubectl apply -f config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Build the docker image
docker-build: test
	docker build . -t ${IMG}:${TAG}

# Deploy ccp-istio-operator on k8s cluster using kustomize
# https://book.kubebuilder.io/quick-start.html#installation
os ?= $(shell go env GOOS)
arch ?= $(shell go env GOARCH)
deploy-kustomize:
	curl -o kustomize -sL https://go.kubebuilder.io/kustomize/${os}/${arch}
	chmod 777 ./kustomize
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}:${TAG}"'@' ./config/default/manager_image_patch.yaml
	# eval $(minikube docker-env)
	./kustomize build config/default | kubectl apply -f -
	# kubectl logs ccp-istio-operator-controller-manager-0 -n=ccp-istio-operator-system -c=manager
	kubectl get all --all-namespaces | grep ccp-istio-operator
	rm -rf ./kustomize

delete-kustomize:
	curl -o kustomize -sL https://go.kubebuilder.io/kustomize/${os}/${arch}
	chmod 777 ./kustomize
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}:${TAG}"'@' ./config/default/manager_image_patch.yaml
	-./kustomize build config/default | kubectl delete -f -
	-kubectl get all --all-namespaces | grep ccp-istio-operator
	rm -rf ./kustomize

# Push the docker image
docker-push:
	docker push ${IMG}:${TAG}

# Clean docker image and other binaries
clean:
	-docker images --format "{{.ID}} {{.Repository}}" | \
	  grep '<none>\|ccp-istio-operator\|golang\|gcr.io/distroless/static' | \
	  awk '{print $1}' | xargs docker rmi
	rm -rf ./manager
	rm -rf kubebuilder_2.0.0-alpha.1_linux_amd64*
	rm -rf ./kustomize

##################################################################################################
# Code generation make targets, use them only if needed, usually not needed for dev, test and CI #
##################################################################################################

# Generate manifests e.g. CRD, RBAC etc.
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
manifests: controller-gen
        $(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./api/...;./controllers/..." output:crd:artifacts:config=config/crd/bases

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
