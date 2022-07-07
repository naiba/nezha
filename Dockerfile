FROM ubuntu:latest

ARG TARGETPLATFORM
ENV TZ="Asia/Shanghai"

COPY ./script/entrypoint.sh /entrypoint.sh

RUN export DEBIAN_FRONTEND="noninteractive" &&
    apt update && apt install -y ca-certificates tzdata &&
    update-ca-certificates &&
    ln -fs /usr/share/zoneinfo/$TZ /etc/localtime &&
    dpkg-reconfigure tzdata &&
    chmod +x /entrypoint.sh

WORKDIR /dashboard
COPY ./resource ./resource
COPY target/$TARGETPLATFORM/dashboard ./app

VOLUME ["/dashboard/data"]
EXPOSE 80 5555
ENTRYPOINT ["/entrypoint.sh"]