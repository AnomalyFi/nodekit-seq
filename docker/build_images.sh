#!/usr/bin/env bash

# Build local Avalanche nodes Docker images using Packer
# The images are tagged with the AvalancheGo version, VM name and VM version
# The images are pushed to the local Docker registry
# Usage: ./build_images.sh
# Environment variables:
# - AVALANCHEGO_VERSION: the AvalancheGo version to build (default: 1.10.10)
# - AVALANCHEGO_VM_NAME: the name of the blockchain VM to install on the node (default: tokenvm)
# - AVALANCHEGO_VM_VERSION: the version of the blockchain VM to install on the node (default: 0.0.999)

set -euo pipefail

# Set default values for environment variables
AVALANCHEGO_VERSION=${AVALANCHEGO_VERSION:-1.10.10}
AVALANCHEGO_VM_NAME=${AVALANCHEGO_VM_NAME:-tokenvm}
AVALANCHEGO_VM_VERSION=${AVALANCHEGO_VM_VERSION:-0.0.999}

# Init Packer
packer init packer/avalanche-node-docker.pkr.hcl

# Build the Avalanche node base image
packer build -var "avalanchego_version=${AVALANCHEGO_VERSION}" \
	-var "avalanchego_vm_name=${AVALANCHEGO_VM_NAME}" \
	-var "avalanchego_vm_version=${AVALANCHEGO_VM_VERSION}" \
	packer/avalanche-node-docker.pkr.hcl

# Build the local Avalanche nodes images
for v in {1..5}; do
	packer build -var "avalanchego_version=${AVALANCHEGO_VERSION}" \
		-var "avalanchego_vm_name=${AVALANCHEGO_VM_NAME}" \
		-var "avalanchego_vm_version=${AVALANCHEGO_VM_VERSION}" \
		-var "avalanche_node_name=validator0${v}" \
		packer/avalanche-node-local-docker.pkr.hcl
done

