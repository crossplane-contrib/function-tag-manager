# Configuration Azure Network

This is a sample composition for testing the function with Azure resources. The composition creates Azure networking resources including a Resource Group, Virtual Network, and Subnets to test the function's handling of tags on Azure resources.

This example runs on Crossplane version 2.0 and higher as it creates Namespaced resources.

## Rendering the Composition

Run the `./render.sh` command in this directory.

```shell
./render.sh
```

Or use the full crossplane render command:

```shell
crossplane render xr.yaml composition.yaml functions.yaml --observed-resources observed-resources --include-full-xr
```

## Updated CompositeResourceDefinition

The [xrd](xrd.yaml) has been updated with the following fields:

- `spec.parameters.ignoreTagKeysReplace`. A list of tag keys to ignore if set by an external system
- `spec.parameters.ignoreTagKeysRetain`. A list of tag keys to ignore if set by an external system. If the tag is defined on the desired resource the external value is ignored
- `spec.parameters.additionalTags`. A map of additional tags to add to each resource
- `spec.parameters.optionalTags`. A map of additional tags to add to each resource only if the resource doesn't have a different value

## Skipping Resources

Applying the following label to a resource will cause the function to skip managing tags on that resource:

```yaml
metadata:
  labels:
     tag-manager.crossplane.io/ignore-resource: true
```

## Azure-Specific Notes

### Tag Support

Azure resources support tags, but not all resource types accept tags. This function automatically filters resources based on the generated filter in `filters/zz_provider-upjet-azure.go`:

- Resource Groups: ✅ Support tags
- Virtual Networks: ✅ Support tags
- Subnets: ✅ Support tags
- Network Security Groups: ✅ Support tags

Resources that don't support tags will be automatically excluded from tag management.

### API Groups

This example supports both namespaced and non-namespaced Azure API groups:

- Cluster=scoped: `azure.upbound.io`
- Namespaced: `azure.m.upbound.io`

## Observed Resources

In the [observed-resources](observed-resources) directory are resources that have had additional tags added for testing purposes to simulate external tag modifications.

## Prerequisites

1. Install Crossplane and the Azure provider:

   ```shell
   kubectl apply -f provider.yaml
   ```

2. Create Azure credentials secret in the resource namespace (update with your credentials):

   ```shell
   kubectl create secret generic azure-creds \
     -n default \
     --from-file=creds=./azure-credentials.json
   ```

3. Create a `ProviderConfig` that matches the namespace of the `Secret`:

  ```shell
  cat <<'EOF' | kubectl apply -f -
  apiVersion: azure.m.upbound.io/v1beta1
  kind: ProviderConfig
  metadata:
    name: default
    namespace: default 
  spec:
    credentials:
      source: Secret
      secretRef:
        name: azure-creds
        namespace: upbound-system
        key: creds
  EOF
  ```
  
4. Install the required functions:

   ```shell
   kubectl apply -f functions.yaml
   ```

5. Create the Composite Resource

  ```shell
  kubectl apply -f xr.yaml
  ```

6. Validate the Composition:

```shell
$ crossplane beta trace network.azure.platform.upbound.io/ref-azure-network
NAME                                                         SYNCED   READY   STATUS
Network/ref-azure-network (default)                          True     True    Available
├─ ResourceGroup/ref-azure-network-b96719dee069 (default)    True     True    Available
├─ Subnet/ref-azure-network-from-xr-sn (default)             True     True    Available
└─ VirtualNetwork/ref-azure-network-from-xr-vnet (default)   True     True    Available
```

7. Verify the tags on the resources

```shell
$ kubectl get virtualnetwork.network.azure.m.upbound.io/ref-azure-network-from-xr-vnet  -o yaml | yq .status.atProvider.tags
romValue: tagFromValue
replaceFromXR: value1
replaceFromXR2: value2
retainFromXR: optional1
retainFromXR2: optional2
```

Try updating the [xr.yaml](xr.yaml) file and the Resources in the Azure portal to
see how tags are managed.

## Example Tag Configuration

The [xr.yaml](xr.yaml) demonstrates:

- **Additional Tags**: Tags applied with `Replace` policy (default)

  ```yaml
  additionalTags:
    replaceFromXR: value1
    replaceFromXR2: value2
  ```

- **Optional Tags**: Tags applied with `Retain` policy

  ```yaml
  optionalTags:
    retainFromXR: optional1
    retainFromXR2: optional2
  ```

- **Ignore Tag Keys (Replace)**: External tags to ignore completely

  ```yaml
  ignoreTagKeysReplace:
  - ignoreReplace1
  - ignoreReplace2
  ```

- **Ignore Tag Keys (Retain)**: External tags to keep if no conflict

  ```yaml
  ignoreTagKeysRetain:
  - ignoreRetain1
  ```

## Resources Created

This composition creates the following Azure resources:

1. **Resource Group**: Container for all networking resources
2. **Virtual Network**: Azure VNet with specified address space
3. **Subnets**: Multiple subnets for different availability zones and access types (public/private)
4. **Network Security Groups**: Security groups for controlling network traffic
5. **Route Tables**: Custom routing for private subnets

All resources that support tags will have the tag management function applied automatically.
