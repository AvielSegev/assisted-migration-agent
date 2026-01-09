# Host


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **string** | Unique identifier for this host | [optional] [default to undefined]
**vendor** | **string** |  | [default to undefined]
**model** | **string** |  | [default to undefined]
**cpuCores** | **number** | Number of CPU cores | [optional] [default to undefined]
**cpuSockets** | **number** | Number of CPU sockets | [optional] [default to undefined]
**memoryMB** | **number** | Host memory in MB | [optional] [default to undefined]

## Example

```typescript
import { Host } from 'migration-agent-api-client';

const instance: Host = {
    id,
    vendor,
    model,
    cpuCores,
    cpuSockets,
    memoryMB,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
