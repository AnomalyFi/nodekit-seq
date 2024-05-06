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

variable "avalanche_node_name" {
  type    = string
  default = "validator01"
}

variable "avalanche_node_groups" {
  type = list(string)
  default = [
    "avalanche_nodes",
    "bootstrap_nodes",
  ]
}

source "docker" "avalanche_node" {
  image  = "ash/avalanche-node:${var.avalanchego_version}-${var.avalanchego_vm_name}-${var.avalanchego_vm_version}"
  pull   = false
  commit = true
  changes = [
    "ENTRYPOINT /opt/avalanche/avalanchego/current/avalanchego --plugin-dir=/opt/avalanche/avalanchego/current/plugins --config-file=/etc/avalanche/avalanchego/mounted-conf/node.json"
  ]
}

build {
  name = "avalanche-node-local-${var.avalanche_node_name}"
  sources = [
    "source.docker.avalanche_node"
  ]

  provisioner "ansible" {
    ansible_env_vars = [
      "ANSIBLE_CONFIG=./ansible.cfg",
      "SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt"
    ]
    inventory_directory = "./inventory/local"
    host_alias          = "${var.avalanche_node_name}"
    groups              = var.avalanche_node_groups
    playbook_file       = "./ansible_collections/ash/avalanche/playbooks/provision_nodes.yml"
    extra_arguments = [
      "--tags",
      "config-avalanchego,config-subnets,config-chains,config-ash_cli",
      "--extra-vars",
      "{avalanchego_auto_restart: false, avalanchego_public_ip: ''}",
      "--extra-vars",
      "{avalanchego_version: ${var.avalanchego_version}}",
      "--extra-vars",
      "{avalanchego_vms_install: { \"${var.avalanchego_vm_name}\": \"${var.avalanchego_vm_version}\" } }"
    ]
  }

  post-processor "docker-tag" {
    repository = "ash/avalanche-node-local-${var.avalanche_node_name}"
    tags       = ["${var.avalanchego_version}-${var.avalanchego_vm_name}-${var.avalanchego_vm_version}"]
  }
}
