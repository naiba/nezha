# !/bin/sh
xgo -v -targets=linux/amd64 -dest release -out nezha-$1 -pkg cmd/$1/main.go .
