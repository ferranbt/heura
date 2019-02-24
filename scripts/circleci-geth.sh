#!/usr/bin/env bash

set -o errexit

VERSION="1.8.23-c9427004"
OS="linux"

DOWNLOAD=https://gethstore.blob.core.windows.net/builds/geth-${OS}-amd64-${VERSION}.tar.gz

install() {
    curl -L $DOWNLOAD > geth.tar.gz

    tar -xvf ./geth.tar.gz -C /tmp/
    mv /tmp/geth-${OS}-amd64-${VERSION}/geth /usr/local/bin/geth
    chmod +x /usr/local/bin/geth
}

run() {
    nohup geth -dev -rpc &
}

# install_geth

"$@"
