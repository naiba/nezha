#/bin/bash

builddir=build/${GOOS}_${GOARCH}
mkdir -p $builddir
echo "BUILDDIR=$builddir" >> $GITHUB_ENV

if [[ $GOOS = "windows" ]]; then
    if [[ $GOARCH = "amd64" ]]; then
	gocc=x86_64-w64-mingw32-gcc
    elif [[ $GOARCH = "386" ]]; then
	gocc=i686-w64-mingw32-gcc
    fi
    CC=${gocc} go build -ldflags "-s -w --extldflags '-static -fpic' -X github.com/naiba/nezha/service/singleton.Version=${VERSION}" -o ${builddir}/dashboard-windows-${GOARCH}.exe -trimpath ./cmd/dashboard
else
    gocc=$(echo -n "${TOOLCHAIN}" | sed 's/-[^-]*$/-gcc/')
    CC=${gocc} go build -ldflags "-s -w --extldflags '-static -fpic' -X github.com/naiba/nezha/service/singleton.Version=${VERSION}" -o ${builddir}/dashboard-${GOOS}-${GOARCH} -trimpath ./cmd/dashboard
fi
