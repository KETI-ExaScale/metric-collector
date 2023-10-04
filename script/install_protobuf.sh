#!/bin/bash

apt update

apt install wget unzip

wget https://github.com/protocolbuffers/protobuf/releases/download/v22.2/protoc-22.2-linux-x86_64.zip

unzip protoc-22.2-linux-x86_64.zip

cp bin/protoc /usr/local/bin/protoc
cp -r include/google /usr/local/include/google

rm -rf bin
rm -rf include
rm -rf protoc-22.2-linux-x86_64.zip
rm -rf readme.txt

#go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
#go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

#echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> /root/.bashrc

export PATH="$PATH:$(go env GOPATH)/bin"

#cat /root/.bashrc



