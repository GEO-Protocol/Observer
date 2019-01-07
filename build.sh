#!/usr/bin/env bash
go build -ldflags "-X main.BuildTimestamp=`date -u '+%Y-%m-%d-%I:%M:%S%p'` -X main.GitHash=`git rev-parse HEAD`" -o bin/observers/0/observer
