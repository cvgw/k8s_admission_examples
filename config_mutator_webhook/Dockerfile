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

RUN mkdir -p /golang/src/config-mutator
WORKDIR /golang/src/config-mutator

COPY Gopkg.lock Gopkg.toml ./
COPY vendor ./vendor

COPY main.go ./

RUN dep check
RUN go build

FROM ubuntu:16.04
RUN apt-get update
COPY --from=builder /golang/src/config-mutator/config-mutator /config-mutator
CMD ["/config-mutator"]
