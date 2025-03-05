# Configuration AWS Network

This is a sample composition for testing the function, based on Upbound's [configuration-aws-network](https://github.com/upbound/configuration-aws-network). The composition creates a number of EC2 resources for us to
test the function's handling of tags.

## Rendering the Composition

Run the `./render.sh` command in this directory.

```shell
./render.sh
```

```shell
crossplane render xr.yaml composition.yaml functions.yaml --observed-resources observed-resources --include-full-xr
```

## Updated CompositeResourceDefinition

The [xrd](xrd.yaml) has been updated with the following fields:

- `spec.parameters.ignoreTagKeysReplace`. A list of tag keys to ignore if set by an external system
- `spec.parameters.ignoreTagKeysRetain`. A list of tag keys to ignore if set by an external system. If the tag is defined on the desired resource the external value is ignored
- `spec.paramaters.additionalTags`. A map of additional tags to add to each resources
- `spec.paramaters.optionalTags`. A map of additional tags to add to each resources only if the resource doesn't have a different value

## Skipping Resources

Applying the following label to a resource will cause the function to skip managing tags on that resource:

```yaml
metadata:
  labels:
     tag-manager.crossplane.io/ignore-resource: true
```

## Observed Resources

In the [observed-resource](observed-resources) directory are resources that have had
additional tags added for testing.
