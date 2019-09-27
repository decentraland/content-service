FROM golang:1.12.0
WORKDIR /go/src/github.com/decentraland/content-service/
ENV GO111MODULE=on
ENV GOPATH=/go
ENV GOCACHE=/root/.cache/go-build
ENV GOOS=linux
COPY . .
RUN go get
RUN make build

EXPOSE 8000
CMD ["./build/content"]