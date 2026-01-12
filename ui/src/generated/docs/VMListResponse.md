# VMListResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**vms** | [**Array&lt;VM&gt;**](VM.md) |  | [default to undefined]
**total** | **number** | Total number of VMs matching the filter | [default to undefined]
**page** | **number** | Current page number | [default to undefined]
**pageCount** | **number** | Total number of pages | [default to undefined]

## Example

```typescript
import { VMListResponse } from 'migration-agent-api-client';

const instance: VMListResponse = {
    vms,
    total,
    page,
    pageCount,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
