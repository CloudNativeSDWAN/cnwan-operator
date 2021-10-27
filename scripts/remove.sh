#!/bin/bash

# This script removes the CN-WAN Operator
function print_error {
  echo && echo 'An error occurred while removing the operator'
  exit 1
}
function print_success {
  echo && echo 'CN-WAN Operator removed successfully'
  exit 0
}
trap print_error ERR

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PARENT_DIR=$(dirname $DIR)
DEPLOY_DIR=$PARENT_DIR/artifacts/deploy
if [ "$(ls -A $DEPLOY_DIR/other)" ]; then
    echo "removing resources from 'other' directory..."
    kubectl delete -f $DEPLOY_DIR/other
fi;

# The name of the deployment has been moved to just "cnwan-operator", so these
# two lines delete both the old name and the new one and make sure to not exit
# the operator in case they were not found.
# Note that we don't remove secrets here: since we remove the whole namespace
# at the end, they will be removed as well.
# TODO: remove this v0.8.0 or v0.9.0
kubectl delete deployment cnwan-operator-controller-manager -n cnwan-operator-system || true
kubectl delete deployment cnwan-operator -n cnwan-operator-system || true
kubectl delete rolebinding cnwan-operator-rolebinding -n cnwan-operator-system
kubectl delete role cnwan-operator-role -n cnwan-operator-system
kubectl delete clusterrolebinding cnwan-operator-cluster-rolebinding
kubectl delete clusterrole cnwan-operator-cluster-role
kubectl delete serviceaccount cnwan-operator-service-account -n cnwan-operator-system
kubectl delete configmap cnwan-operator-settings -n cnwan-operator-system
kubectl delete namespace cnwan-operator-system

print_success
