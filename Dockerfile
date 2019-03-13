# Use the offical Golang image to create a build artifact.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang:1.10 as builder

# Copy local code to the container image.
WORKDIR /go/src/github.com/knative/docs/helloworld
COPY . .

RUN cd /usr/local/go/

COPY ./knative-eventing/vendor /go/src/

RUN go get github.com/knative/eventing/pkg/provisioners
RUN go get github.com/knative/pkg/signals
RUN go get go.uber.org/zap
RUN go get sigs.k8s.io/controller-runtime/pkg/client/config
RUN go get sigs.k8s.io/controller-runtime/pkg/manager

RUN rm -rf /go/src/github.com/knative/eventing/vendor/*



# Build the outyet command inside the container.
# (You may fetch or manage dependencies here,
# either manually or with a tool like "godep".)
RUN CGO_ENABLED=0 GOOS=linux go build -v -o helloworld

# Use a Docker multi-stage build to create a lean production image.
# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
FROM alpine

# Copy the binary to the production image from the builder stage.
COPY --from=builder /go/src/github.com/knative/docs/helloworld/helloworld /helloworld

# Service must listen to $PORT environment variable.
# This default value facilitates local development.
ENV PORT 8080

# Run the web service on container startup.
CMD ["/helloworld"]