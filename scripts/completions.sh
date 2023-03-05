#!/bin/sh
# MIT Licensed
# Copyright (c) 2023 Roberto García <roberto@planta7.io>

set -e
rm -rf completions
mkdir completions
for sh in bash zsh fish; do
	go run main.go completion "$sh" > "completions/serve.$sh"
done
