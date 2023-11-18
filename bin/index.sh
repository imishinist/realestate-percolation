#!/bin/bash

INDEX_NAME=$1
file=$2

if [ -z "$file" ]; then
  echo "empty file args" >&2
  exit 1
fi

if [ ! -f "$file" ]; then
  echo "file not exist" >&2
  exit 1
fi

while read -r line; do
  curl -X POST \
    -H "Content-Type: application/json" \
    -u elastic:password \
    -k http://localhost:9200/$INDEX_NAME/_doc -d "$line" | jq
done < $file
