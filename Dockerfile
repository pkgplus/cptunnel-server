## build stage
FROM golang:1.14.4-alpine3.12 as build-env

# repo
RUN cp /etc/apk/repositories /etc/apk/repositories.bak
RUN echo "http://mirrors.aliyun.com/alpine/v3.12/main/" > /etc/apk/repositories
RUN echo "http://mirrors.aliyun.com/alpine/v3.12/community/" >> /etc/apk/repositories

# git
RUN apk update
RUN apk add --no-cache git

# move to GOPATH
RUN mkdir -p /app
WORKDIR /app

# go mod
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod .
COPY go.sum .

# build
COPY . .
RUN go build -o /app/cmdplus-tunnel-server server.go

## docker image stage
FROM alpine:3.12

# repo
RUN cp /etc/apk/repositories /etc/apk/repositories.bak
RUN echo "http://mirrors.aliyun.com/alpine/v3.12/main/" > /etc/apk/repositories
RUN echo "http://mirrors.aliyun.com/alpine/v3.12/community/" >> /etc/apk/repositories

# timezone
RUN apk update
RUN apk add --no-cache tzdata bash curl \
    && echo "Asia/Shanghai" > /etc/timezone \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

## Add Tini
#RUN apk add --no-cache tini
#ENTRYPOINT ["/sbin/tini", "--"]

COPY --from=build-env /app /app

EXPOSE 80
WORKDIR /app
CMD ["/app/cmdplus-tunnel-server"]
