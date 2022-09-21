#!/bin/bash

if [ -z "$1" ]
  then
    echo "Please provide version in the format v0.0.0"
    exit 1
fi
go mod tidy
git tag "$1"
git push origin "$1"
