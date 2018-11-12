FROM golang:1.11.0 as builder
WORKDIR /go/src/github.com/decentraland/content-service/
ENV GO111MODULE=on
ENV GOPATH=/go
ENV GOCACHE=/root/.cache/go-build
ENV GOOS=linux
COPY . .
RUN go get
RUN go build

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/decentraland/content-service/content-service .
EXPOSE 8000
CMD ["./content-service"]
