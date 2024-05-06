#!/usr/bin/env bash
# Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
# See the file LICENSE for licensing terms.

set -o errexit
set -o nounset
set -o pipefail

# Set the CGO flags to use the portable version of BLST
#
# We use "export" here instead of just setting a bash variable because we need
# to pass this flag to all child processes spawned by the shell.
export CGO_CFLAGS="-O -D__BLST_PORTABLE__"

# Root directory
TOKENVM_PATH=$(
    cd "$(dirname "${BASH_SOURCE[0]}")"
    cd .. && pwd
)

realpath() {
    [[ $1 = /* ]] && echo "$1" || echo "$PWD/${1#./}"
}

if [[ $# -eq 1 ]]; then
    BINARY_PATH=$(realpath $1)
elif [[ $# -eq 0 ]]; then
    # Set default binary directory location
    name="tokenvm"
    BINARY_PATH=$TOKENVM_PATH/build/$name
else
    echo "Invalid arguments to build tokenvm. Requires zero (default location) or one argument to specify binary location."
    exit 1
fi

cd $TOKENVM_PATH

echo "Building tokenvm in $BINARY_PATH"
mkdir -p $(dirname $BINARY_PATH)
go build -o $BINARY_PATH ./cmd/tokenvm

CLI_PATH=$TOKENVM_PATH/build/token-cli
echo "Building token-cli in $CLI_PATH"
mkdir -p $(dirname $CLI_PATH)
go build -o $CLI_PATH ./cmd/token-cli

FAUCET_PATH=$TOKENVM_PATH/build/token-faucet
echo "Building token-faucet in $FAUCET_PATH"
mkdir -p $(dirname $FAUCET_PATH)
go build -o $FAUCET_PATH ./cmd/token-faucet

FEED_PATH=$TOKENVM_PATH/build/token-feed
echo "Building token-feed in $FEED_PATH"
mkdir -p $(dirname $FEED_PATH)
go build -o $FEED_PATH ./cmd/token-feed

# Pack the binaries into a Ash-compatible tarball
AVALANCHEGO_VM_VERSION=${AVALANCHEGO_VM_VERSION:-0.0.999}
echo "Packing binaries into a Ash-compatible tarball tokenvm_"$AVALANCHEGO_VM_VERSION"_linux_amd64.tar.gz"
cd "$TOKENVM_PATH/build"
tar -czf tokenvm_"$AVALANCHEGO_VM_VERSION"_linux_amd64.tar.gz *
echo "Creating checksums file tokenvm_"$AVALANCHEGO_VM_VERSION"_checksums.txt"
sha256sum tokenvm_"$AVALANCHEGO_VM_VERSION"_linux_amd64.tar.gz > tokenvm_"$AVALANCHEGO_VM_VERSION"_checksums.txt
