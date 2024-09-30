#!/bin/sh

crossplane render \
  --observed-resources=observed-resources \
  --include-full-xr \
  --include-function-results \
  xr.yaml composition.yaml functions.yaml