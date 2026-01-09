# Inventory


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**vcenterId** | **string** | ID of the vCenter | [default to undefined]
**clusters** | [**{ [key: string]: InventoryData; }**](InventoryData.md) | Map of cluster names to their inventory data | [default to undefined]
**vcenter** | [**InventoryData**](InventoryData.md) |  | [optional] [default to undefined]

## Example

```typescript
import { Inventory } from 'migration-agent-api-client';

const instance: Inventory = {
    vcenterId,
    clusters,
    vcenter,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
