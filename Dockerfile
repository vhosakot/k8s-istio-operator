# Build the manager binary
FROM golang:1.12.5 as builder

WORKDIR /go/src/wwwin-github.cisco.com/CPSG/ccp-istio-operator
# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY vendor/ vendor/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:latest
WORKDIR /
COPY --from=builder /go/src/wwwin-github.cisco.com/CPSG/ccp-istio-operator/manager .
ENTRYPOINT ["/manager"]
