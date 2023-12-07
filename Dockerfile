FROM alpine:edge

ARG TARGETOS
ARG TARGETARCH

RUN apk update && \
    apk upgrade --no-cache && \
    apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo 'Asia/Shanghai' >/etc/timezone && \
    rm -rf /var/cache/apk/*

COPY ./script/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

WORKDIR /dashboard
COPY dist/dashboard-${TARGETOS}-${TARGETARCH} ./app

VOLUME ["/dashboard/data"]
EXPOSE 80 5555
ENTRYPOINT ["/entrypoint.sh"]