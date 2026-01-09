# VMs


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**total** | **number** |  | [default to undefined]
**totalMigratable** | **number** |  | [default to undefined]
**totalMigratableWithWarnings** | **number** |  | [optional] [default to undefined]
**cpuCores** | [**VMResourceBreakdown**](VMResourceBreakdown.md) |  | [default to undefined]
**diskSizeTier** | [**{ [key: string]: DiskSizeTierSummary; }**](DiskSizeTierSummary.md) |  | [optional] [default to undefined]
**diskTypes** | [**{ [key: string]: DiskTypeSummary; }**](DiskTypeSummary.md) |  | [optional] [default to undefined]
**distributionByCpuTier** | **{ [key: string]: number; }** | Distribution of VMs across CPU tier buckets (e.g., \&quot;0-4\&quot;, \&quot;5-8\&quot;, \&quot;9-16\&quot;, \&quot;17-32\&quot;, \&quot;32+\&quot;) | [optional] [default to undefined]
**distributionByMemoryTier** | **{ [key: string]: number; }** | Distribution of VMs across Memory tier buckets (e.g., \&quot;0-4\&quot;, \&quot;5-16\&quot;, \&quot;17-32\&quot;, \&quot;33-64\&quot;, \&quot;65-128\&quot;, \&quot;129-256\&quot;, \&quot;256+\&quot;) | [optional] [default to undefined]
**distributionByNicCount** | **{ [key: string]: number; }** | Distribution of VMs by NIC count (e.g., \&quot;0\&quot;, \&quot;1\&quot;, \&quot;2\&quot;, \&quot;3\&quot;, \&quot;4+\&quot;) | [optional] [default to undefined]
**ramGB** | [**VMResourceBreakdown**](VMResourceBreakdown.md) |  | [default to undefined]
**diskGB** | [**VMResourceBreakdown**](VMResourceBreakdown.md) |  | [default to undefined]
**diskCount** | [**VMResourceBreakdown**](VMResourceBreakdown.md) |  | [default to undefined]
**nicCount** | [**VMResourceBreakdown**](VMResourceBreakdown.md) |  | [optional] [default to undefined]
**powerStates** | **{ [key: string]: number; }** |  | [default to undefined]
**os** | **{ [key: string]: number; }** |  | [optional] [default to undefined]
**osInfo** | [**{ [key: string]: OsInfo; }**](OsInfo.md) |  | [optional] [default to undefined]
**notMigratableReasons** | [**Array&lt;MigrationIssue&gt;**](MigrationIssue.md) |  | [default to undefined]
**migrationWarnings** | [**Array&lt;MigrationIssue&gt;**](MigrationIssue.md) |  | [default to undefined]

## Example

```typescript
import { VMs } from 'migration-agent-api-client';

const instance: VMs = {
    total,
    totalMigratable,
    totalMigratableWithWarnings,
    cpuCores,
    diskSizeTier,
    diskTypes,
    distributionByCpuTier,
    distributionByMemoryTier,
    distributionByNicCount,
    ramGB,
    diskGB,
    diskCount,
    nicCount,
    powerStates,
    os,
    osInfo,
    notMigratableReasons,
    migrationWarnings,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
