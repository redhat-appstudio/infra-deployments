#!/bin/bash

# print goes to stderr hence the redirections
opa eval --data ./data 'print(data)' 2>&1 >/dev/null | yq e -P -
