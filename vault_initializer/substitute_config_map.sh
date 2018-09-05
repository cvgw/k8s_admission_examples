#!/bin/bash
set -e

(echo "cat <<EOF"; cat ./kubernetes/config-map.yml; echo EOF) | sh
