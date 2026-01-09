# Infra


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**totalHosts** | **number** |  | [default to undefined]
**totalDatacenters** | **number** |  | [optional] [default to undefined]
**totalClusters** | **number** |  | [optional] [default to undefined]
**clustersPerDatacenter** | **Array&lt;number&gt;** |  | [optional] [default to undefined]
**cpuOverCommitment** | **number** | CPU Overcommitment Ratio. Calculated as total Allocated vCPUs / Total Physical Cores | [optional] [default to undefined]
**memoryOverCommitment** | **number** | RAM memory Overcommitment Ratio. Calculated as total Allocated memory / Total memory available | [optional] [default to undefined]
**hosts** | [**Array&lt;Host&gt;**](Host.md) |  | [optional] [default to undefined]
**hostsPerCluster** | **Array&lt;number&gt;** |  | [optional] [default to undefined]
**vmsPerCluster** | **Array&lt;number&gt;** |  | [optional] [default to undefined]
**hostPowerStates** | **{ [key: string]: number; }** |  | [default to undefined]
**networks** | [**Array&lt;Network&gt;**](Network.md) |  | [default to undefined]
**datastores** | [**Array&lt;Datastore&gt;**](Datastore.md) |  | [default to undefined]

## Example

```typescript
import { Infra } from 'migration-agent-api-client';

const instance: Infra = {
    totalHosts,
    totalDatacenters,
    totalClusters,
    clustersPerDatacenter,
    cpuOverCommitment,
    memoryOverCommitment,
    hosts,
    hostsPerCluster,
    vmsPerCluster,
    hostPowerStates,
    networks,
    datastores,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
