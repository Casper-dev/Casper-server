#!/bin/bash

./thrift -gen go casperproto.thrift
rm -rf casperproto
mv gen-go/casperproto casperproto/
rm -rf gen-go
