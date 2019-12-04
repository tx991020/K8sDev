#!/bin/bash
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o k8sdep .;
docker build -f ./Dockerfile . -t registry.cn-hangzhou.aliyuncs.com/guanghe/k8sdep:$1;
docker push registry.cn-hangzhou.aliyuncs.com/guanghe/k8sdep:$1;