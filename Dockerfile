# Build the manager binary
FROM golang:1.12.9 as builder

WORKDIR /go/src/wwwin-github.cisco.com/CPSG/ccp-istio-operator
COPY vendor/ vendor/
# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

RUN echo 'IPCentral information: Base image is debian:9.9-slim'
FROM debian:9.9-slim

RUN apt-get update \
    && apt-get install --no-install-recommends -y wget \
    # install helm needed to install istio
    && wget --no-check-certificate https://storage.googleapis.com/kubernetes-helm/helm-v2.12.2-linux-amd64.tar.gz \
    && tar -zxvf helm-v2.12.2-linux-amd64.tar.gz \
    && mv linux-amd64/helm /usr/local/bin/helm \
    && rm -rf linux-amd64 helm-v2.12.2-linux-amd64.tar.gz \
    && echo "Helm version:" \
    && helm version --client \
    # remove unwanted stuff in container
    && apt-get remove -y --purge wget \
    && apt-get -y clean all \
    && apt-get -y autoclean \
    && apt-get -y autoremove \
    && rm -rf /var/lib/apt/lists/* \
    && rm -rf /var/cache/apt \
    # print IPCentral information
    && echo 'IPCentral information: Start debian packages' \
    && cat /etc/apt/sources.list \
    && dpkg --list \
    && echo 'IPCentral information: End debian packages'

WORKDIR /
COPY --from=builder /go/src/wwwin-github.cisco.com/CPSG/ccp-istio-operator/manager .
ENTRYPOINT ["/manager"]
