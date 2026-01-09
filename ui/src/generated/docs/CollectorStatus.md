# CollectorStatus


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**status** | **string** |  | [default to undefined]
**hasCredentials** | **boolean** | Whether vCenter credentials are configured | [default to undefined]
**error** | **string** | Error message when status is error | [optional] [default to undefined]

## Example

```typescript
import { CollectorStatus } from 'migration-agent-api-client';

const instance: CollectorStatus = {
    status,
    hasCredentials,
    error,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
