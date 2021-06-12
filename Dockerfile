FROM golang:latest

LABEL maintainer="<nyg1991@aliyun.com>"

LABEL version="1.0"

ARG GOPROXY=https://goproxy.io

ENV GOPROXY ${GOPROXY}
ENV GO111MODULE on

WORKDIR /go/src/

ADD src/ .

RUN go mod download
RUN go build server.go
RUN mkdir /go/app
RUN mkdir /go/app/logs
RUN mv server /go/app/

WORKDIR /go/app

ADD cmd/conf.json .

EXPOSE 80