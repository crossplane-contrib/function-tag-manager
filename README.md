# function-manage-tags

`function-manage-tags` is a [Crossplane](https://crossplane.io) function that allows
Platform Operators to manage Cloud tags on managed resources.

There several use cases for this Function:

- Allowing external systems to set tags on Crossplane Managed Resources without conflict.
- Adding Common Tags to Resources without having to update every resource in a Composition.
- Allowing users the ability to add their own tags when Requesting new resources.

## Installing the Function

Please note, during early testing this function is located at my personal repository. It
will move to crossplane-contrib once initial testing is successful.

```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: Function
metadata:
  name: borrelli-org-function-tag-manager
spec:
  package: xpkg.upbound.io/borrelli-org/function-tag-manager:v0.1.0
```

## Using this Function in a Composition

This function is designed to be a step in a [Composition Pipeline](https://docs.crossplane.io/latest/concepts/compositions/#use-a-pipeline-of-functions-in-a-composition) after other functions have
created Desired State. Below is an example pipeline step:

```yaml
- step: manage-tags
  functionRef:
    name: borrelli-org-function-tag-manager
  input:
    apiVersion: tag-manger.fn.crossplane.io/v1beta1
    kind: ManagedTags
    addTags:
    - type: FromValue
      policy: Replace
      tags: 
        from: value
        add: tags
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
      keys:
      - ignoreReplace1
      - ignoreReplace2
    - type: FromCompositeFieldPath
      fromFieldPath: spec.parameters.ignoreTagKeysReptain
      policy: Retain
      keys:
      - ignoreRetain1
```

## Function Inputs

### AddTags

The `addTags` field configures tags that will be added to every resource.

The `FromValue` type indicates that `tags` will be defined in the function input.

The `FromCompositeField` type indicates that the tags will be imported from the Composite Resource manifest.

```yaml
   addTags:
    - type: FromValue
      policy: Replace
      tags: 
        from: value
        add: tags
    - type: FromCompositeFieldPath
      fromFieldPath: spec.parameters.additionalTags
      policy: Replace
    - type: FromCompositeFieldPath
      fromFieldPath: spec.parameters.optionalTags
      policy: Retain
```

### IgnoreTags

The `ignoreTags` configures Observed tags in the Cloud that Crossplane will "ignore". In most
cases, Crossplane will attempt to manage every field of a resource, and if a difference is calculated
Crossplane will update the resource and remove fields that are not in the Desired state.

There are many Cloud management systems that set tags on Resources. By adding the keys
to those tags in the `ignoreTags` section, the function will populate the Desired state with
the values of the Observed tags for each key defined.

Tag keys to ignore can be defined in `FromValue` or set in the Composite/Claim using `FromCompositeFieldPath`.

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
      keys:
      - ignoreReplace1
      - ignoreReplace2
    - type: FromCompositeFieldPath
      fromFieldPath: spec.parameters.ignoreTagKeysRetain
      policy: Retain
      keys:
      - ignoreRetain1

```

Another option for allowing external systems to manage tags is to use the [`initProvider`](https://docs.crossplane.io/latest/concepts/managed-resources/#initprovider) field of a Managed Resource.

## Tag Policies

When Merging tags, a `Policy` can be set:

- `Replace` (default) in the case the desired and observed tags don't match, the observed value will replace desired.
- `Retain` in the case the desired and observed tags don't match, the desired value will remain.

## Skipping Resources Manually

This function will skip any resource with the `tag-manager.fn.crossplane.io/ignore-resource` Kubernetes label set to `True` or `true`:

```yaml
apiVersion: ec2.aws.upbound.io/v1beta1
kind: InternetGateway
metadata:
  labels:
    tag-manager.fn.crossplane.io/ignore-resource: "True"
  name: my-igw
```

## Filtering Resources

The initial version of this function support simple filters.

### Managed Resource Filter

If the resource does not have a `spec.forProvider` field it skipped.

### API Group Filter

If the resources does not contain one of the following API groups, it is skipped:

- `aws.upbound.io`
- `azure.upbound.io`
- `gcp.upbound.io`

### Crossplane Resources

Crossplane Resources like ProviderConfigs are skipped.

## Developing this Function

```shell
# Run code generation - see input/generate.go
$ go generate ./...

# Run tests
$ go test -cover ./...
ok      github.com/stevendborrelli/function-tag-manager 0.398s  coverage: 67.9% of statements
        github.com/stevendborrelli/function-tag-manager/input/v1beta1           coverage: 0.0% of statements

# Lint the code
$ docker run --rm -v $(pwd):/app -v ~/.cache/golangci-lint/v1.61.0:/root/.cache -w /app golangci/golangci-lint:v1.61.0 golangci-lint run

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
docker build . --tag=function-unit-test-runtime
```

Next create the Crossplane Package:

```shell
crossplane xpkg build -f package --embed-runtime-image=function-unit-test-runtime -o function-tag-manager.xpkg
```
