#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

#TODO add a golang installer here
# Download prometheus
rm -f /tmp/prometheus
wget https://github.com/prometheus/prometheus/releases/download/v2.43.0/prometheus-2.43.0.linux-amd64.tar.gz
tar -xvf prometheus-2.43.0.linux-amd64.tar.gz
rm prometheus-2.43.0.linux-amd64.tar.gz
mv prometheus-2.43.0.linux-amd64/prometheus /tmp/prometheus
rm -rf prometheus-2.43.0.linux-amd64

# sudo apt install build-essential
# wget https://go.dev/dl/go1.20.11.linux-amd64.tar.gz
# sudo tar -C /usr/local -xvf go1.20.11.linux-amd64.tar.gz
# export PATH=$PATH:/usr/local/go/bin



# Import chains and demo.pk key
#
# Assumes token-cli has already been transferred into the machine
/tmp/token-cli chain import-ops aops.yml
/tmp/token-cli key import demo.pk

# Start prometheus server
tmux new-session -d -s prometheus '/tmp/token-cli prometheus generate --prometheus-open-browser=false --prometheus-start=true'
