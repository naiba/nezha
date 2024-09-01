#!/bin/bash


err() {
    echo "$*" >&2
}

get_musl() {
    case "$GOARCH" in
	"amd64")
	    toolchain="x86_64-linux-musl-native"
	    ;;
	"arm")
	    toolchain="armv7l-linux-musleabihf-cross"
	    ;;
	"arm64")
	    toolchain="aarch64-linux-musl-cross"
	    ;;
	*)
	    toolchain="${GOARCH}-linux-musl-cross"
	    ;;
    esac
    wget -qP /tmp https://musl.cc/${toolchain}.tgz
    sudo mkdir /opt/toolchains >/dev/null 2>&1;
    sudo tar -zxf /tmp/${toolchain}.tgz -C /opt/toolchains
    echo "TOOLCHAIN=$toolchain" >> $GITHUB_ENV
    echo "/opt/toolchains/${toolchain}/bin" >> $GITHUB_PATH
}

get_mingw() {
    sudo apt install mingw-w64
}

case "$GOOS" in
    "linux")
	get_musl
	;;
    "windows")
	get_mingw
	;;
    *)
	err "Unknown OS; exiting..."
	exit 1
	;;
esac
