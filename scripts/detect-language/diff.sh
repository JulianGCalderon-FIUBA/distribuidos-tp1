#!/usr/bin/env bash

: "${N:=100}"

delta -s \
      <(head -n "$N" english_reviews_go.csv) \
      <(head -n "$N" english_reviews_py.csv)
