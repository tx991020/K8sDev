FROM alpine:3.4

RUN apk update && apk add --no-cache ca-certificates && \
    apk add tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

ADD ./k8sdep /go/bin/k8sdep
COPY config /go/src/k8sdep/config
WORKDIR /go/src/k8sdep
CMD ["/go/bin/k8sdep"]
EXPOSE 5001
