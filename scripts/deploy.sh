#!/bin/bash

# This script deploys the CN-WAN Operator
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
  echo && echo 'CN-WAN Operator deployed successfully'
  exit 0
}
trap print_error ERR

function print_help {
    echo "Usage:"
    echo "deploy.sh servicedirectory|etcd [--help|options]"
    echo
    echo
    echo "Global options:"
    echo "--image       the image repository, in case you don't want to use CN-WAN Operator's default one"
    echo "--help        show this help"
    echo
    echo "servicedirectory options:"
    echo "--service-account     the path to the google service account (default: ./artifacts/secrets/gcloud-credentials.json)."
    echo
    echo "etcd options:"
    echo "--username    the username for etcd. This will also be used to create the corresponding Kubernetes secrets."
    echo "--password    the password for etcd. This will also be used to create the corresponding Kubernetes secrets."
    echo
    echo "Examples:"
    echo "deploy.sh etcd --username user --password pass"
    echo "deploy.sh servicedirectory --image example.com/username/repo:tag"
    echo
}

function no_sr_err {
    echo "error: no or invalid service registry provided."
    echo
    print_help
    exit 1
}

if [ "$#" -lt 1 ]; then
    no_sr_err
fi;

SR=""
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PARENT_DIR=$(dirname $DIR)
DEPLOY_DIR=$PARENT_DIR/artifacts/deploy
SECRETS_DIR=$PARENT_DIR/artifacts/secrets
SERVICE_ACCOUNT_PATH=$SECRETS_DIR/gcloud-credentials.json
IMG=""
ETCD_USERNAME=""
ETCD_PASSWORD=""

# Does deploy exist?
dir_exists $DEPLOY_DIR
if [ "$DIR_VALID" -eq 0 ]; then
    echo "deploy folder does not exist"
    exit 1
fi;

# Does settings exist?
SETTINGS_DIR=$PARENT_DIR/artifacts/settings
dir_exists $SETTINGS_DIR
if [ "$DIR_VALID" -eq 0 ]; then
    echo "error: settings folder does not exist"
    exit 1
fi;

# Does the settings file exist?
SETTINGS_YAML=$SETTINGS_DIR/settings.yaml
file_exists $SETTINGS_YAML
if [ "$FILE_VALID" -eq 0 ]; then
    echo "error: settings file does not exist in $SETTINGS_DIR"
    exit 1
fi;

# Parse the flags
if [ "$1" == "--help" ]; then
    print_help
    exit 0
fi;

if [ \( "$1" != "servicedirectory" \) -a \( "$1" != "etcd" \) ]; then
    no_sr_err
fi;
SR=$1
shift

while test $# -gt 0; do
    case "$1" in
        --username)
            shift
            if test $# -gt 0; then
                ETCD_USERNAME=$1
            else
                echo "error: no username provided"
                exit 1
            fi
            shift
        ;;

        --password)
            shift
            if test $# -gt 0; then
                ETCD_PASSWORD=$1
            else
                echo "error: no password provided"
                exit 1
            fi
            shift
        ;;

        --service-account)
            shift
            if test $# -gt 0; then
                SERVICE_ACCOUNT_PATH=$1
            else
                echo "error: no service account path"
                exit 1
            fi
            shift
        ;;

        --image)
            shift
            if test $# -gt 0; then
                IMG=$1
            else
                echo "error: no image provided"
                exit 1
            fi
            shift
        ;;

        *)
            break
        ;;
    esac
done

# Check for gcloud service account existence
if [ "$SR" == "servicedirectory" ]; then
    # if [ "$1" == "ServiceDirectory" ]; then
        # SR="SD"
        # Does the service account file exist?
        file_exists $SERVICE_ACCOUNT_PATH
        if [ "$FILE_VALID" -eq 0 ]; then
            echo "error: $SERVICE_ACCOUNT_PATH file was not found"
            exit 1
        fi;
    # fi;
fi;

if [ "$SR" == "etcd" ]; then
    if [ \( ! -z "$ETCD_USERNAME" \) -a \( -z "$ETCD_PASSWORD" \)  ]; then
        echo "error: no password provided"
        exit 1
    fi;
    if [ \( ! -z "$ETCD_PASSWORD" \) -a \( -z "$ETCD_USERNAME" \)  ]; then
        echo "error: no username provided"
        exit 1
    fi;
fi;

if [ -z "$IMG" ]; then
    TAG=$($DIR/get-latest.sh)
    IMG=cnwan/cnwan-operator:$TAG
fi;

echo "using $IMG"
echo "all files found, deploying..."

kubectl create -f $DEPLOY_DIR/01_namespace.yaml
kubectl create -f $DEPLOY_DIR/02_service_account.yaml,$DEPLOY_DIR/03_cluster_role.yaml,$DEPLOY_DIR/04_cluster_role_binding.yaml,$DEPLOY_DIR/05_role.yaml,$DEPLOY_DIR/06_role_binding.yaml
kubectl create configmap cnwan-operator-settings -n cnwan-operator-system --from-file=$SETTINGS_YAML

if [ "$SR" == "servicedirectory" ]; then
    kubectl create secret generic google-service-account -n cnwan-operator-system --from-file=$SERVICE_ACCOUNT_PATH
fi;

if [ "$SR" == "etcd" ]; then
    if [ \( ! -z "$ETCD_PASSWORD" \) -a \( ! -z "$ETCD_USERNAME" \) ]; then
        kubectl create secret generic etcd-credentials -n cnwan-operator-system --from-literal=username=$ETCD_USERNAME --from-literal=password=$ETCD_PASSWORD
    fi;
fi;

sed -e "s#{CONTAINER_IMAGE}#$IMG#" $DEPLOY_DIR/07_deployment.yaml.tpl > $DEPLOY_DIR/07_deployment_generated.yaml
kubectl create -f $DEPLOY_DIR/07_deployment_generated.yaml

print_success