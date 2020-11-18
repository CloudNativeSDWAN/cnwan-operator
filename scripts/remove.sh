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

kubectl delete deployment cnwan-operator-controller-manager -n cnwan-operator-system
kubectl delete clusterrolebinding cnwan-operator-manager-rolebinding
kubectl delete clusterrole cnwan-operator-manager-role
kubectl delete serviceaccount cnwan-operator-service-account -n cnwan-operator-system
kubectl delete secret cnwan-operator-service-handler-account -n cnwan-operator-system
kubectl delete configmap cnwan-operator-settings -n cnwan-operator-system
kubectl delete namespace cnwan-operator-system

print_success
