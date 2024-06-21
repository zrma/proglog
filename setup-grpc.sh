#!/usr/bin/env bash

set -e

GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)

if [[ $GOOS == 'linux' && $GOARCH == 'amd64' ]]; then
    platform='linux-x86_64'
elif [[ $GOOS == 'linux' && $GOARCH == 'arm64' ]]; then
    platform='linux-aarch_64'
elif [[ $GOOS == 'darwin' && $GOARCH == 'amd64' ]]; then
    platform='osx-x86_64'
elif [[ $GOOS == 'darwin' && $GOARCH == 'arm64' ]]; then
    platform='osx-aarch_64'
elif [[ $GOOS == 'windows' ]]; then
    platform='win64'
else
    echo "unsupported platform GOOS=$GOOS GOARCH=$GOARCH"
    exit 1
fi

protocVersion=25.1
protocName="protoc-${protocVersion}-${platform}"

protocDir="./tools/${protocName}"
protocBin="${protocDir}/bin/protoc"

if [[ ! -f "$protocBin" ]]; then
    protocUrlPrefix="https://github.com/protocolbuffers/protobuf/releases"
    protocUrl="${protocUrlPrefix}/download/v${protocVersion}/${protocName}.zip"
    protocDownloadPath="${protocDir}.zip"

    echo "Downloading $protocUrl..."
    curl -L $protocUrl -o $protocDownloadPath

    echo "Unzipping $protocDownloadPath..."
    mkdir -p $protocDir
    unzip $protocDownloadPath -d $protocDir
    rm -f $protocDownloadPath
else
    echo "$protocBin exists."
fi

go install \
    'google.golang.org/grpc/cmd/protoc-gen-go-grpc' \
    'google.golang.org/protobuf/cmd/protoc-gen-go' \
    'github.com/bufbuild/buf/cmd/buf'
