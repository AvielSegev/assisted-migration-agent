# InspectionStatus


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**state** | **string** | Current inspection state | [default to undefined]
**error** | **string** | Error message when state is error | [optional] [default to undefined]
**results** | **object** | Inspection results | [optional] [default to undefined]

## Example

```typescript
import { InspectionStatus } from 'migration-agent-api-client';

const instance: InspectionStatus = {
    state,
    error,
    results,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
