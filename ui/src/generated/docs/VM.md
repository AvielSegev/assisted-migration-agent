# VM


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **string** | VM name | [default to undefined]
**id** | **string** | VM ID | [default to undefined]
**vCenterState** | **string** | vCenter state (e.g., green, yellow, red) | [default to undefined]
**datacenter** | **string** | Datacenter name | [default to undefined]
**cluster** | **string** | Cluster name | [default to undefined]
**diskSize** | **string** | Total disk size (e.g., 12GB) | [default to undefined]
**memory** | **string** | Memory size (e.g., 16GB) | [default to undefined]
**issues** | **Array&lt;string&gt;** | List of issues found during inspection | [default to undefined]
**inspection** | [**InspectionStatus**](InspectionStatus.md) |  | [default to undefined]

## Example

```typescript
import { VM } from 'migration-agent-api-client';

const instance: VM = {
    name,
    id,
    vCenterState,
    datacenter,
    cluster,
    diskSize,
    memory,
    issues,
    inspection,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
