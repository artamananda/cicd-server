#!/bin/bash

GOOS=linux GOARCH=arm64 go build -o ../build/cicd-server-arm64 ../main.go
