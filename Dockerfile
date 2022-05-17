FROM golang:alpine AS binarybuilder
RUN apk --no-cache --no-progress add \
    gcc git musl-dev
WORKDIR /dashboard
COPY . .
RUN cd cmd/dashboard && go build -o app -ldflags="-s -w"

FROM alpine:latest
ENV TZ="Asia/Shanghai"
COPY ./script/entrypoint.sh /entrypoint.sh
RUN apk --no-cache --no-progress add \
    ca-certificates \
    tzdata && \
    cp "/usr/share/zoneinfo/$TZ" /etc/localtime && \
    echo "$TZ" >  /etc/timezone && \
    chmod +x /entrypoint.sh
WORKDIR /dashboard
COPY ./resource ./resource
COPY --from=binarybuilder /dashboard/cmd/dashboard/app ./app

VOLUME ["/dashboard/data"]
EXPOSE 80 5555
ENTRYPOINT ["/entrypoint.sh"]