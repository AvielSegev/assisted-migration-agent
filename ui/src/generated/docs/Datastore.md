# Datastore


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**type** | **string** |  | [default to undefined]
**totalCapacityGB** | **number** |  | [default to undefined]
**freeCapacityGB** | **number** |  | [default to undefined]
**vendor** | **string** |  | [default to undefined]
**diskId** | **string** |  | [default to undefined]
**hardwareAcceleratedMove** | **boolean** |  | [default to undefined]
**protocolType** | **string** |  | [default to undefined]
**model** | **string** |  | [default to undefined]
**hostId** | **string** | Identifier of the host where this datastore is attached | [optional] [default to undefined]

## Example

```typescript
import { Datastore } from 'migration-agent-api-client';

const instance: Datastore = {
    type,
    totalCapacityGB,
    freeCapacityGB,
    vendor,
    diskId,
    hardwareAcceleratedMove,
    protocolType,
    model,
    hostId,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
