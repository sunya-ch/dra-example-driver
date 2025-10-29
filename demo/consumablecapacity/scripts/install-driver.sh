#!/usr/bin/env bash

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This scripts invokes `kind build image` so that the resulting
# image has a containerd with CDI support.
#
# Usage: install-driver.sh <resource file> <resource number>

# A reference to the current directory where this script is located

CURRENT_DIR="$(cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd)"
ROOT_DIR="$(cd "${CURRENT_DIR}/../../../" && pwd)"

set -ex
set -o pipefail

: ${RS_NUM_DEVICE:=8}

# Set Helm chart value file
VALUES_FILE=${ROOT_DIR}/deployments/helm/dra-example-driver/values.yaml

if [ -n "${RS_CONFIG}" ]; then
  RS_CONFIG_FILE=${ROOT_DIR}/demo/consumablecapacity/config/${RS_CONFIG}.yaml
  VALUES_FILE=${ROOT_DIR}/demo/consumablecapacity/generated_${RS_CONFIG}_values.
  yq eval ".kubeletPlugin.configMap.data.\"slice-config.yaml\" \
    = load_str(\"${RS_CONFIG_FILE}\") \
    | .kubeletPlugin.configMap.enable = true" \
    ${ROOT_DIR}/deployments/helm/dra-example-driver/values.yaml > ${VALUES_FILE}
fi

if [ ! -f "${VALUES_FILE}" ]; then
  echo "Error: Values File '${VALUES_FILE}' does not exist." >&2
  exit 1
fi

helm upgrade -i \
  --create-namespace \
  --namespace dra-example-driver \
  dra-example-driver \
  deployments/helm/dra-example-driver \
  --set kubeletPlugin.numDevices=${RS_NUM_DEVICE} \
  -f ${VALUES_FILE}