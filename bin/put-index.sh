#!/bin/bash

INDEX_NAME=$1
CONFIG_FILE=$2

if [ -z "$INDEX_NAME" ]; then
    echo "empty index" >&2
    exit 1
fi
if [ -z "$CONFIG_FILE" ]; then
    echo "empty config file" >&2
    exit 1
fi
if [ ! -f "$CONFIG_FILE" ]; then
    echo "config file not exists" >&2
    exit 1
fi

curl -X PUT \
    -H "Content-Type: application/json" \
    -u elastic:password \
    -k https://localhost:9200/$INDEX_NAME -d@${CONFIG_FILE}
