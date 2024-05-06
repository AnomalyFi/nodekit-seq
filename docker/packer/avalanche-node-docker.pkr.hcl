packer {
  required_plugins {
    docker = {
      version = ">= 1.0.1"
      source  = "github.com/hashicorp/docker"
    }
    ansible = {
      version = ">= 1.1.0"
      source  = "github.com/hashicorp/ansible"
    }
  }
}

variable "avalanchego_version" {
  type    = string
  default = "1.10.10"
}

variable "avalanchego_vm_name" {
  type    = string
  default = "tokenvm"
}

variable "avalanchego_vm_version" {
  type    = string
  default = "0.0.999"
}

source "docker" "ubuntu" {
  image  = "ubuntu:22.04"
  commit = true
}

build {
  name = "avalanche-node-${var.avalanchego_version}-${var.avalanchego_vm_name}-${var.avalanchego_vm_version}"
  sources = [
    "source.docker.ubuntu"
  ]

  provisioner "shell" {
    inline = [
      "apt-get update",
      "apt-get install -y ca-certificates gpg python3 sudo",
    ]
  }

  provisioner "ansible" {
    ansible_env_vars = [
      "ANSIBLE_CONFIG=./ansible.cfg",
      "SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt"
    ]
    inventory_directory = "./inventory/local"
    host_alias          = "avalanche_node"
    groups = [
      "avalanche_nodes",
      "bootstrap_nodes"
    ]
    playbook_file = "./ansible_collections/ash/avalanche/playbooks/provision_nodes.yml"
    extra_arguments = [
      "--tags",
      "install-avalanchego,install-vms,install-ash_cli",
      "--extra-vars",
      "{avalanchego_auto_restart: false}",
      "--extra-vars",
      "{avalanchego_version: ${var.avalanchego_version}}",
      "--extra-vars",
      "{avalanchego_vms_install: { \"${var.avalanchego_vm_name}\": \"${var.avalanchego_vm_version}\" } }"
    ]
  }

  post-processor "docker-tag" {
    repository = "ash/avalanche-node"
    tags       = ["${var.avalanchego_version}-${var.avalanchego_vm_name}-${var.avalanchego_vm_version}"]
  }
}
