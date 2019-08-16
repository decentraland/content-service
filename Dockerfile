FROM golang:1.12.0
WORKDIR /go/src/github.com/decentraland/content-service/
ENV GO111MODULE=on
ENV GOPATH=/go
ENV GOCACHE=/root/.cache/go-build
ENV GOOS=linux
COPY . .
RUN go get
RUN GIT_COMMIT=$(git rev-list -1 HEAD) && \
      go build -ldflags "-X github.com/decentraland/dcl-gin/pkg/dclgin.version=$GIT_COMMIT"

EXPOSE 8000
CMD ["./content-service"]