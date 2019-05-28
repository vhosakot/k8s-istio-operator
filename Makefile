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
	  # kubectl apply -f helm/crd.yaml
	  # ./bin/manager

# Run ccp-istio-operator go binary against the configured Kubernetes cluster in ~/.kube/config
run-binary: fmt vet
	# create ccp-istio-operator CRD
	kubectl apply -f helm/crd.yaml
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
	# use locally built docker image in helm/deployment.yaml
	sed -i'' -e 's@image: .*@image: '"${IMG}:${TAG}"'@' helm/deployment.yaml
	# update imagePullPolicy to "Never" in helm/deployment.yaml to
	# pull locally built docker image into k8s pod
	sed -i'' -e 's@imagePullPolicy: .*@imagePullPolicy: 'Never'@' helm/deployment.yaml
	kubectl apply -f ./helm/
	kubectl get all --all-namespaces | grep ccp-istio-operator
	kubectl get crd | grep istios.operator.ccp.cisco.com

# Delete ccp-istio-operator on k8s
delete-k8s:
	-kubectl delete -f ./helm/

# Build docker image
docker-build: test
	# if using minikube for dev, run:
	#  eval $(minikube docker-env)
	docker build . -t ${IMG}:${TAG}

# Push docker image
docker-push:
	docker push ${IMG}:${TAG}

# Delete docker image, ccp-istio-operator on k8s and other binaries
clean:
	-docker images --format "{{.ID}} {{.Repository}}" | \
	  grep '<none>\|ccp-istio-operator\|golang\|gcr.io/distroless/static' | \
	  awk '{print $1}' | xargs docker rmi
	-kubectl delete -f ./helm/
	rm -rf ./bin
	rm -rf ./kubebuilder_2.0.0-alpha.1_linux_amd64*
	rm -rf ./kustomize
	rm -rf ./cover.out

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
