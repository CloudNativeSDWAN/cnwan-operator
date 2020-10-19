#!/bin/bash

# This script prints the latest version of the CNWAN Operator

ORG=cnwan
REPO=cnwan-operator
JSON=$(curl -s https://hub.docker.com/v2/repositories/$ORG/$REPO/tags/)
TAGS_LIST=$( echo $JSON | grep -o '"name"\s*:\s*"[^"]*"')

for i in ${TAGS_LIST}
do
    TAG=$( echo $i | cut -c8- | tr -d '"' )
    if [[ $TAG == v* ]]; then
        echo $TAG
        exit 0
    fi;
done