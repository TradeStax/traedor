#!/bin/bash

go build -o bin/traedor

if [ $? -ne 0 ]; then
  exit 1
fi

./bin/traedor
