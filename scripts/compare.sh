#!/usr/bin/env bash

if [ "$#" -ne 2 ]; then
  echo "Compare results with reference values"
  echo
  echo "Usage: $0 <results-dir> <reference-dir>"
  exit 1
fi
ACTUAL=$1
REFERENCE=$2
FAILURE=0

section() {
  echo
  echo "============"
  echo "Comparing Q$1"
  echo "============"
  echo
}

section 1

if diff --color=always \
  "$ACTUAL/1.csv" \
  "$REFERENCE/1.csv"
then
  echo "OK!"
  cat "$ACTUAL/1.csv"
else
  FAILURE=1
fi

section 2

if diff --color=always \
  "$ACTUAL/2.csv" \
  "$REFERENCE/2.csv"
then
  echo "OK!"
  cat "$ACTUAL/2.csv"
else
  FAILURE=1
fi

section 3

if diff --color=always \
  "$ACTUAL/3.csv" \
  "$REFERENCE/3.csv"
then
  echo "OK!"
  cat "$ACTUAL/3.csv"
else
  FAILURE=1
fi

section 4

if diff --color=always \
  <(sort "$ACTUAL/4.csv") \
  <(sort "$REFERENCE/4.csv")
then
  echo "OK!"
  cat "$ACTUAL/4.csv"
else
  FAILURE=1
fi

section 5

if diff --color=always \
  <(sort "$ACTUAL/5.csv") \
  <(sort "$REFERENCE/5.csv")
then
  echo "OK!"
  echo "Output is too long, showing first lines:"
  head "$ACTUAL/5.csv"
else
  FAILURE=1
fi

exit "$FAILURE"
