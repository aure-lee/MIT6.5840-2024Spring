#!/bin/bash

go build -buildmode=plugin ../mrapps/wc.go

rm -f mr-*