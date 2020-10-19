#!/bin/bash

# This script deploys the CNWAN Operator
DIR_VALID=0
dir_exists () {
    if [ -d "$1" ]; then
        DIR_VALID=1
    else
        DIR_VALID=0
    fi;
}

FILE_VALID=0
file_exists () {
    if [ -f "$1" ]; then
        FILE_VALID=1
    else
        FILE_VALID=0
    fi;
}

function print_error {
  echo && echo 'An error occurred while deploying'
  exit 1
}
function print_success {
  echo && echo 'CNWAN Operator deployed successfully'
  exit 0
}
trap print_error ERR

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PARENT_DIR=$(dirname $DIR)
DEPLOY_DIR=$PARENT_DIR/deploy

IMG=$1
if [ -z "$IMG" ]; then
    TAG=$($DIR/get-latest.sh)
    IMG=cnwan/cnwan-operator:$TAG
fi;

echo "using $IMG"

# Does deploy exist?
dir_exists $DEPLOY_DIR
if [ "$DIR_VALID" -eq 0 ]; then
    echo "deploy folder does not exist"
    exit 1
fi;

# Does settings exist?
SETTINGS_DIR=$DEPLOY_DIR/settings
dir_exists $SETTINGS_DIR
if [ "$DIR_VALID" -eq 0 ]; then
    echo "settings folder does not exist"
    exit 1
fi;

# Does the settings file exist?
SETTINGS_YAML=$SETTINGS_DIR/settings.yaml
file_exists $SETTINGS_YAML
if [ "$FILE_VALID" -eq 0 ]; then
    echo "settings file does not exist in $SETTINGS_DIR"
    exit 1
fi;

# Does the service account file exist?
SERV_ACC=$SETTINGS_DIR/gcloud-credentials.json
file_exists $SERV_ACC
if [ "$FILE_VALID" -eq 0 ]; then
    echo "service account file does not exist in $SETTINGS_DIR"
    exit 1
fi;

echo "all files found, deploying..."
kubectl create -f $DEPLOY_DIR/01_namespace.yaml
kubectl create configmap cnwan-operator-settings -n cnwan-operator-system --from-file=$SETTINGS_YAML
kubectl create secret generic cnwan-operator-service-handler-account -n cnwan-operator-system --from-file=$SERV_ACC
kubectl create -f $DEPLOY_DIR/02_service_account.yaml,$DEPLOY_DIR/03_cluster_role.yaml,$DEPLOY_DIR/04_cluster_role_binding.yaml
sed -e "s#{CONTAINER_IMAGE}#$IMG#" $DEPLOY_DIR/05_deployment.yaml.tpl > $DEPLOY_DIR/05_deployment_generated.yaml && kubectl create -f $DEPLOY_DIR/05_deployment_generated.yaml

print_success