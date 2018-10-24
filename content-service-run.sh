#!/bin/bash
export  GOPATH=/go
export  GOCACHE=/root/.cache/go-buil
echo " --------------------- "
echo "| Building....        |"
echo " --------------------- "
go build
echo " --------------------- "
echo "| Running....         |"
echo " --------------------- "
./content-service
