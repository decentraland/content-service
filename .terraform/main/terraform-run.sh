#!/bin/bash
REGION=$1
ENV=$2

rm -rfv .terraform terraform.tfstate.backup

terraform init \
  -backend-config=../config/${REGION}/${ENV}/default.backend \
  -var-file=../config/${REGION}/${ENV}/default.tfvars

if test $? -ne 0; then
  echo "Unable ter perform terraform init"
  exit 2;
fi

#terraform destroy -auto-approve \
#-var-file=../config/$REGION/${ENV}/default.backend \
#-var-file=../config/$REGION/${ENV}/default.tfvars
#  if test $? -ne 0; then
#    echo "Unable ter perform terraform destroy"
#    exit 2;
#fi

terraform apply -auto-approve \
    -var-file=../config/$REGION/${ENV}/default.backend \
    -var-file=../config/$REGION/${ENV}/default.tfvars
