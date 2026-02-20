#!/bin/sh

crossplane render \
  --extra-resources environmentConfigs.yaml \
  --observed-resources=observed-resources \
  --include-full-xr \
  --include-function-results \
  xr.yaml composition.yaml functions.yaml

