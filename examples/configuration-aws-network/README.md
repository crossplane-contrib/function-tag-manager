# Configuration AWS Network

This is a sample composition for testing the function, based on Upbound's [configuration-aws-network](https://github.com/upbound/configuration-aws-network). The composition creates a number of EC2 resources for us to
test the function's handling of tags.

## Rendering the Composition

Run the `./render.sh` command in this directory.

```shell
./render.sh
```

```shell
crossplane render network-xrd.yaml composition.yaml functions.yaml --observed-resources observed-resources --include-full-xr
```

## Updated CompositeResourceDefinition

The [xrd](xrd.yaml) has been updated with the following 2 fields:

- `spec.parameters.ignoretags`. A list of tag keys to ignore
- `spec.paramaters.additionalTags`. A map of additional tags to add to each resources

## Skipping resources 

Applying the following label to a resource will cause the function to skip managing tags on that resource:

```yaml
metadata:
  labels:
     tag-manager.crossplane.io/ignore-resource: true
```

## Observed Resources

In the [observed-resource](observed-resources) directory are resources that have had
additional tags added for testing. 