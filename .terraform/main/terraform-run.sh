#!/bin/bash
REGION=$1
ENV=$2
BRANCH=$3
PROJECT=$4
LASTCOMMIT=$5
CONTAINER_DEFINITION_FILE="container-definition.json"

echo " ------------------------------------- "
echo "| Using commit number: ${LASTCOMMIT} "
echo " ------------------------------------  "

cd ../config/${REGION}/${ENV}
sed "s/tag-number/${LASTCOMMIT}/g" ${CONTAINER_DEFINITION_FILE} > tmp.json
mv tmp.json ${CONTAINER_DEFINITION_FILE}
cat ${CONTAINER_DEFINITION_FILE}
cd ${WORKSPACE}/${PROJECT}/.terraform/main


rm -rfv .terraform terraform.tfstate.backup

terraform init \
  -backend-config=../config/${REGION}/${ENV}/default.backend \
  -var-file=../config/${REGION}/${ENV}/default.tfvars

if test $? -ne 0; then
  echo "Unable to perform terraform init"
  exit 2;
fi

terraform destroy -auto-approve \
-var-file=../config/$REGION/${ENV}/default.backend \
-var-file=../config/$REGION/${ENV}/default.tfvars
  if test $? -ne 0; then
    echo "Unable ter perform terraform destroy"
    exit 2;
fi



case $BRANCH in
  master|development)
            terraform apply -auto-approve \
            -var-file=../config/$REGION/${ENV}/default.backend \
            -var-file=../config/$REGION/${ENV}/default.tfvars
  ;;

  *)
           echo " ------------------------------------- "
           echo "| Not deploy, only plan!!....         |"
            echo " ------------------------------------ "
            terraform plan \
            -var-file=../config/$REGION/${ENV}/default.backend \
            -var-file=../config/$REGION/${ENV}/default.tfvars
  ;;
esac
