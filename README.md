# function-tag-manager

`function-tag-manager` is a [Crossplane](https://crossplane.io) function that allows
Platform Operators to manage Cloud tags on managed resources.

AWS and Azure resources managed by upjet-based providers that support tags are
supported, and the function can manage tags for both cluster and namespace-scoped
resources.

There several use cases for this Function:

- Allowing external systems to set tags on Crossplane Managed Resources without conflict.
- Adding Common Tags to Resources without having to update every resource in a Composition.
- Allowing users the ability to add their own tags when Requesting new resources.
- Removing tags that have been set earlier in the pipeline by other functions.

## Installing the Function

The function is installed as a Crossplane Package. Apply the following
YAML manifest to your Crossplane cluster.

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: crossplane-contrib-function-tag-manager
spec:
  package: xpkg.upbound.io/crossplane-contrib/function-tag-manager:v0.6.0
```

## Using this Function in a Composition

This function is designed to be a step in a [Composition Pipeline](https://docs.crossplane.io/latest/concepts/compositions/#use-a-pipeline-of-functions-in-a-composition) after other functions have
created Desired State. Below is an example pipeline step:

```yaml
- step: manage-tags
  functionRef:
    name: crossplane-contrib-function-tag-manager
  input:
    apiVersion: tag-manager.fn.crossplane.io/v1beta1
    kind: ManagedTags
    addTags:
    - type: FromValue
      policy: Replace
      tags: 
        key1: value1
        key2: value2
    - type: FromCompositeFieldPath
      fromFieldPath: spec.parameters.additionalTags
      policy: Replace
    - type: FromCompositeFieldPath
      fromFieldPath: spec.parameters.optionalTags
      policy: Retain
    ignoreTags:
    - type: FromValue
      policy: Replace
      keys:
      - external-tag-1
      - external-tag-2
    - type: FromCompositeFieldPath
      fromFieldPath: spec.parameters.ignoreTagKeysReplace
      policy: Replace
    - type: FromCompositeFieldPath
      fromFieldPath: spec.parameters.ignoreTagKeyRetain
      policy: Retain
```

## Function Inputs

### AddTags

The `addTags` field configures tags that will be added to every resource.

The `FromValue` type indicates that `tags` will be defined in the function input.

The `FromCompositeField` type indicates that the tags will be imported from the Composite Resource manifest.

The `FromEnvironmentFieldPath` type indicates that the tags will be imported from the [Environment](https://docs.crossplane.io/latest/composition/environment-configs/).

```yaml
   addTags:
    - type: FromValue
      policy: Replace
      tags: 
        key1: value1
        key2: value2
    - type: FromCompositeFieldPath
      fromFieldPath: spec.parameters.additionalTags
      policy: Replace
    - type: FromCompositeFieldPath
      fromFieldPath: spec.parameters.optionalTags
      policy: Retain
    - type: FromEnvironmentFieldPath
      fromFieldPath: tags
```

### IgnoreTags

The `ignoreTags` configures Observed tags in the Cloud that Crossplane will "ignore". In most
cases, Crossplane will attempt to manage every field of a resource, and if a difference is calculated
Crossplane will update the resource and remove fields that are not in the Desired state.

There are many Cloud management systems that set tags on Resources. By adding the keys
to those tags in the `ignoreTags` section, the function will populate the Desired state with
the values of the Observed tags for each key defined.

Tag keys to ignore can be defined in `FromValue`, in the Composite/Claim using `FromCompositeFieldPath` or from EnvironmentConfig using `FromEnvironmentFieldPath`

```yaml
ignoreTags:
- type: FromValue
  policy: Replace
  keys:
  - external-tag-1
  - external-tag-2
- type: FromCompositeFieldPath
  fromFieldPath: spec.parameters.ignoreTagKeysReplace
  policy: Replace
- type: FromCompositeFieldPath
  fromFieldPath: spec.parameters.ignoreTagKeysRetain
  policy: Retain
- type: FromEnvironmentFieldPath
  fromFieldPath: ignoreTags
```

Another option for allowing external systems to manage tags is to use the [`initProvider`](https://docs.crossplane.io/latest/concepts/managed-resources/#initprovider) field of a Managed Resource.

### RemoveTags

The function can remove tags defined in the desired state by specifying
`removeTags` and providing an array of keys to delete.

```yaml
  removeTags:
  - type: FromValue
    keys: 
    - fromValue2
  - type: FromCompositeFieldPath
    fromFieldPath: spec.parameters.removeTags
  - type: FromEnvironmentFieldPath
    fromFieldPath: removeTags
```

## Tag Policies

When Merging tags, a `Policy` can be set:

- `Replace` (default) in the case the desired and observed tags don't match, the observed value will replace desired.
- `Retain` in the case the desired and observed tags don't match, the desired value will remain.

## Skipping Resources Manually

This function will skip any resource with the `tag-manager.fn.crossplane.io/ignore-resource` Kubernetes annotation set to `True` or `true`:

```yaml
apiVersion: ec2.aws.upbound.io/v1beta1
kind: InternetGateway
metadata:
  annotations:
    tag-manager.fn.crossplane.io/ignore-resource: "True"
  name: my-igw
```

**Note:** Versions v0.7.0 and earlier of the function used a label instead of the annotation to skip a resource. For backward compatibility, the label `tag-manager.fn.crossplane.io/ignore-resource` is still supported. However, if both the annotation and label are present, the annotation takes precedence. Using annotations is the recommended approach as it follows Kubernetes best practices.

## Filtering Resources

This function supports both AWS and Azure resources that allow setting of tags.

### AWS Resources

A scan of the AWS provider shows that 498 resources support tags and 477 do not. Starting with the 2.x provider both Cluster
and Namespace-scoped resources are supported, so for each resource there are two Custom Resource Definitions.

The AWS Provider CRDs were scanned using [`cmd/generator/main.go`](cmd/generator/main.go) to generate the list in [filters/zz_provider-upjet-aws.go](filters/zz_provider-upjet-aws.go).

### Azure Resources

A scan of the Azure provider shows that 550 resources support tags and 927 do not. Both Cluster-scoped (`azure.upbound.io`)
and Namespace-scoped (`azure.m.upbound.io`) API groups are supported.

The Azure Provider CRDs were scanned using the same generator to create the list in [filters/zz_provider-upjet-azure.go](filters/zz_provider-upjet-azure.go).

### Regenerating Filters

To regenerate the resource filters for both providers:

```shell
cd filters
go generate ./...
```

This will clone the provider repositories, scan their CRDs, and regenerate the filter files.

## Developing this Function

```shell
# Run code generation - see input/generate.go
$ go generate ./...

# Run tests
$ go test -cover ./...
ok      github.com/crossplane-contrib/function-tag-manager      0.542s  coverage: 68.6% of statements
ok      github.com/crossplane-contrib/function-tag-manager/cmd/generator        1.035s  coverage: 43.2% of statements
        github.com/crossplane-contrib/function-tag-manager/cmd/generator/render         coverage: 0.0% of statements
        github.com/crossplane-contrib/function-tag-manager/filters              coverage: 0.0% of statements
        github.com/crossplane-contrib/function-tag-manager/input/v1beta1                coverage: 0.0% of statements

# Lint the code
$ docker run --rm -v $(pwd):/app -v ~/.cache/golangci-lint/v2.6.1:/root/.cache -w /app golangci/golangci-lint:v2.6.1 golangci-lint run
0 issues.

# Build a Docker image - see Dockerfile
$ docker build .
```

## Testing this Function

To test this function, it can be run locally in debug mode:

```shell
# Run your Function in insecure mode
go run . --insecure --debug
```

Once your Function is running, in another window you can use the `render` command.

```shell
# Install Crossplane CLI
$ curl -sL https://raw.githubusercontent.com/crossplane/crossplane/master/install.sh | XP_CHANNEL=stable sh
```

To build the function, run:

```shell
docker build . --tag=function-tag-manager
```

Please note that this command builds an image for your local computer architecture.
In general, Crossplane projects build images for linux/amd64 and linux/arm64.
See the Github [ci.yaml](.github/workflows/ci.yml) workflow for an example.

Next create the Crossplane Package, embedding the function we just built:

```shell
crossplane xpkg build -f package --embed-runtime-image=function-tag-manager -o function-tag-manager.xpkg
```

I use the `up` binary to push to the [Upbound Marketplace](https://marketplace.upbound.io)

```shell
up xpkg push xpkg.upbound.io/crossplane-contrib/function-tag-manager:v0.6.0 -f function-tag-manager.xpkg
```
