FROM ubuntu:16.04 as builder
RUN apt-get update && apt-get install -y curl build-essential unzip git

RUN curl -O https://dl.google.com/go/go1.10.3.linux-amd64.tar.gz
RUN tar -xvf go1.10.3.linux-amd64.tar.gz -C /usr/local
ENV GOROOT=/usr/local/go
ENV PATH=$PATH:/usr/local/go/bin
RUN mkdir -p /golang/src
RUN mkdir -p /golang/bin
ENV GOPATH=/golang
ENV GOBIN=$GOPATH/bin
ENV PATH=$PATH:$GOBIN

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
# End of golang & dep boilerplate

RUN mkdir -p /golang/src/vault_initializer

WORKDIR /golang/src/vault_initializer
COPY main.go .
COPY Gopkg.lock Gopkg.toml ./
COPY vendor ./vendor

RUN dep check
RUN go build

FROM ubuntu:16.04

WORKDIR /tmp
RUN apt-get update && apt-get install -y ca-certificates

WORKDIR /
COPY --from=builder /golang/src/vault_initializer/vault_initializer /

CMD ["/bin/bash", "-c", "./vault_initializer"]
