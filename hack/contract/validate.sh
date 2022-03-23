#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

FORMAT=${1:-pretty}
DATA_DIR=$(dirname $0)/data
POLICY_DIR=$(dirname $0)/policy

opa eval \
  --data $DATA_DIR \
  --data $POLICY_DIR \
  --format $FORMAT \
  data.hacbs.contract.main.deny
