# Setting tags using an EnvironmentConfig

This example is the same as [configuration-aws-network](../configuration-aws-network/), but the tags are set using a Crossplane [EnvironmentConfig](https://docs.crossplane.io/latest/composition/environment-configs/) that can be shared across multiple Compositions.

The outputs of rendering this example and the `configuration-aws-network` should be the same, with the exception that tags are set in the xr.yaml and printed in the output for `configuration-aws-network`.

## Rendering the Composition

Run the `./render.sh` command in this directory.

```shell
./render.sh
```

or run the command directly:

```shell
crossplane render \
  --extra-resources environmentConfigs.yaml \
  --observed-resources=observed-resources \
  --include-full-xr \
  --include-function-results \
  xr.yaml composition.yaml functions.yaml
```

## function-environment-configs

This example uses [function-environment-configs](https://github.com/crossplane-contrib/function-environment-configs) to fetch the
EnvironmentConfigs from the cluster.

## Skipping Resources

Applying the following annotation to a resource will cause the function to skip managing tags on that resource:

```yaml
metadata:
  annotations:
     tag-manager.fn.crossplane.io/ignore-resource: "true"
```

**Note:** For backward compatibility, the label `tag-manager.fn.crossplane.io/ignore-resource` is still supported, but annotations are recommended.

## Observed Resources

In the [observed-resources](observed-resources) directory are resources that have had additional tags added for testing.
