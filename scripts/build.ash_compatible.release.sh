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
SEQVM_PATH=$(
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
    name="seqvm"
    BINARY_PATH=$SEQVM_PATH/build/$name
else
    echo "Invalid arguments to build seqvm. Requires zero (default location) or one argument to specify binary location."
    exit 1
fi

cd $SEQVM_PATH

echo "Building seqvm in $BINARY_PATH"
mkdir -p $(dirname $BINARY_PATH)
go build -o $BINARY_PATH ./cmd/seqvm

CLI_PATH=$SEQVM_PATH/build/seq-cli
echo "Building seq-cli in $CLI_PATH"
mkdir -p $(dirname $CLI_PATH)
go build -o $CLI_PATH ./cmd/seq-cli

# Pack the binaries into a Ash-compatible tarball
AVALANCHEGO_VM_VERSION=${AVALANCHEGO_VM_VERSION:-0.0.999}
echo "Packing binaries into a Ash-compatible tarball seqvm_"$AVALANCHEGO_VM_VERSION"_linux_amd64.tar.gz"
cd "$SEQVM_PATH/build"
tar -czf seqvm_"$AVALANCHEGO_VM_VERSION"_linux_amd64.tar.gz *
echo "Creating checksums file seqvm_"$AVALANCHEGO_VM_VERSION"_checksums.txt"
sha256sum seqvm_"$AVALANCHEGO_VM_VERSION"_linux_amd64.tar.gz > seqvm_"$AVALANCHEGO_VM_VERSION"_checksums.txt
