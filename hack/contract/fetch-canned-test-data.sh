#!/bin/bash
rm -rf ./data
curl -s http://file.bos.redhat.com/~sbaird/hacbs-rego-test-data.tar.gz | tar zxvf -
