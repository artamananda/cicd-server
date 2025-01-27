#!/bin/bash

GOOS=linux GOARCH=arm64 go build -o cicd-server-arm64 main.go
