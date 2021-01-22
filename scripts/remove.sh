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
kubectl delete rolebinding cnwan-operator-rolebinding -n cnwan-operator-system
kubectl delete role cnwan-operator-role -n cnwan-operator-system
kubectl delete clusterrolebinding cnwan-operator-cluster-rolebinding
kubectl delete clusterrole cnwan-operator-cluster-role
kubectl delete serviceaccount cnwan-operator-service-account -n cnwan-operator-system
kubectl delete configmap cnwan-operator-settings -n cnwan-operator-system
kubectl delete namespace cnwan-operator-system

print_success
