# VM Inspector Subsystem - Design Document

## Overview

The VM Inspector subsystem is a new component in the Assisted Migration Agent that performs deep inspection of virtual machines in a VMware vSphere environment. It follows the same asynchronous pattern as the existing Collector service, using a scheduler for background work execution.

### Purpose

The Inspector enables detailed VM analysis by:
- Creating VM snapshots for safe inspection
- Running inspection logic on VMs (placeholder for future deep analysis)
- Managing inspection state per-VM with a queue-based processing model
- Providing real-time status updates via REST API

---

## Architecture

### High-Level Component Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              REST API Layer                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  POST /vms/inspector    - Start inspection                          │    │
│  │  GET  /vms/inspector    - Get inspector status                      │    │
│  │  PATCH /vms/inspector   - Add VMs to queue                          │    │
│  │  DELETE /vms/inspector  - Stop inspector                            │    │
│  │  GET  /vms/{id}/inspector   - Get VM inspection status              │    │
│  │  DELETE /vms/{id}/inspector - Remove VM from queue                  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Handler Layer                                      │
│                      (internal/handlers/vms.go)                             │
│  - Request validation                                                        │
│  - Credential extraction                                                     │
│  - Response mapping (models → API types)                                     │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Service Layer                                       │
│                  (internal/services/inspector.go)                            │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                      InspectorService                                  │  │
│  │  - State machine management                                            │  │
│  │  - Work coordination                                                   │  │
│  │  - Cancellation handling                                               │  │
│  │  - VM queue processing                                                 │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                           │                    │
                           ▼                    ▼
┌────────────────────────────────┐   ┌────────────────────────────────────────┐
│        Scheduler               │   │           Store Layer                   │
│  (pkg/scheduler)               │   │   (internal/store/inspection.go)        │
│  - Work queue execution        │   │   - VM inspection status persistence    │
│  - Concurrent work management  │   │   - Queue ordering (sequence)           │
└────────────────────────────────┘   │   - Filtering & updates                 │
                                     └────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                       Work Builder Layer                                     │
│                   (pkg/vmware/work_builder.go)                               │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                   InsWorkBuilder                                       │  │
│  │  - Init() → vSphere connection                                         │  │
│  │  - Build(moid) → VM inspection work units                              │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         VMware vSphere                                       │
│  - VM snapshot creation/removal                                              │
│  - VM inspection operations                                                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Component Details

### 1. Models (`internal/models/inspector.go`)

#### Inspector State Machine

```
                    ┌─────────┐
                    │  ready  │◄────────────────────────┐
                    └────┬────┘                         │
                         │ Start()                      │
                         ▼                              │
                    ┌─────────┐                         │
                    │ started │                         │
                    └────┬────┘                         │
                         │ Init work complete           │
                         ▼                              │
                    ┌─────────┐                         │
            ┌──────►│ running │◄─────┐                  │
            │       └────┬────┘      │                  │
            │            │           │                  │
            │    ┌───────┴───────┐   │                  │
            │    │               │   │                  │
            │    ▼               ▼   │                  │
            │ VM success    VM error │                  │
            │ (continue)   (continue)│                  │
            │    │               │   │                  │
            │    └───────┬───────┘   │                  │
            │            │           │                  │
            │            │ more VMs? │                  │
            │            └───────────┘                  │
            │                                           │
            │         no more VMs                       │
            │            │                              │
            │            ▼                              │
            │       ┌────────┐                          │
            │       │  done  │──────────────────────────┤
            │       └────────┘                          │
            │                                           │
            │       Cancel()                            │
            │            │                              │
            │            ▼                              │
            │       ┌───────────┐                       │
            └───────│ canceling │                       │
                    └─────┬─────┘                       │
                          │                             │
                          ▼                             │
                    ┌──────────┐                        │
                    │ canceled │────────────────────────┤
                    └──────────┘                        │
                                                        │
                    ┌─────────┐                         │
                    │  error  │─────────────────────────┘
                    └─────────┘
```

**Inspector States (`InspectorState`):**

| State | Description |
|-------|-------------|
| `ready` | Idle, waiting for inspection request |
| `started` | Creating vSphere client connection |
| `running` | Actively inspecting VMs |
| `canceling` | Cancellation in progress |
| `canceled` | Inspection was canceled |
| `done` | All VMs inspected successfully |
| `error` | Fatal error occurred |

**VM Inspection States (`InspectionState`):**

| State | Description |
|-------|-------------|
| `pending` | Queued, waiting for inspection |
| `running` | Currently being inspected |
| `completed` | Inspection finished successfully |
| `canceled` | Removed from queue before processing |
| `error` | Inspection failed for this VM |
| `not_found` | VM not in inspection queue |

#### Key Types

```go
// InspectorStatus - Overall inspector state
type InspectorStatus struct {
    State InspectorState
    Error error
}

// InspectionStatus - Per-VM inspection state
type InspectionStatus struct {
    State InspectionState
    Error error
}

// InspectorWorkBuilder - Interface for building work units
type InspectorWorkBuilder interface {
    Init() InspectorWorkUnit           // Connection/setup work
    Build(string) []InspectorWorkUnit  // Per-VM work units
}

// InspectorWorkUnit - A single unit of inspection work
type InspectorWorkUnit struct {
    Status func() InspectorStatus
    Work   func() func(ctx context.Context) (any, error)
}
```

---

### 2. InspectorService (`internal/services/inspector.go`)

The core service managing the inspection workflow.

#### Key Responsibilities

1. **State Management**: Thread-safe state transitions with mutex protection
2. **Work Coordination**: Orchestrates work builder and scheduler
3. **Queue Processing**: Processes VMs in FIFO order (by sequence)
4. **Error Handling**: Per-VM errors don't stop the inspector; fatal errors do
5. **Cancellation**: Graceful shutdown with context cancellation

#### Public Interface

```go
type InspectorService struct {
    // Dependencies
    scheduler      *scheduler.Scheduler
    store          *store.Store
    builderFactory InspectorWorkBuilderFactory
    
    // State
    status models.InspectorStatus
    mu     sync.Mutex
    done   chan any
    cancel context.CancelFunc
    cred   *models.Credentials
}

// Constructor
func NewInspectorService(s *scheduler.Scheduler, store *store.Store) *InspectorService

// Constructor with custom builder (for testing)
func NewInspectorServiceWithBuilder(s *scheduler.Scheduler, store *store.Store, 
    builderFactory InspectorWorkBuilderFactory) *InspectorService

// Status methods
func (c *InspectorService) GetStatus() models.InspectorStatus
func (c *InspectorService) GetVmStatus(ctx context.Context, moid string) (models.InspectionStatus, error)
func (c *InspectorService) IsBusy() bool

// Control methods
func (c *InspectorService) Start(ctx context.Context, vmIDs []string, cred *models.Credentials) error
func (c *InspectorService) Add(ctx context.Context, vmIDs []string) error
func (c *InspectorService) Stop()
func (c *InspectorService) CancelVmsInspection(ctx context.Context, vmIDs ...string) error
func (c *InspectorService) CancelInspector(ctx context.Context) error
```

#### Start Flow

```
Start(ctx, vmIDs, credentials)
    │
    ├─► Set state: started
    │
    ├─► Create work builder with credentials
    │
    ├─► Execute Init() work unit (vSphere connection)
    │   └─► On error: Set error state, return error
    │
    ├─► Clear existing inspection data (DeleteAll)
    │
    ├─► Add VMs to queue with pending status
    │
    └─► Launch goroutine: run()
            │
            ├─► Set state: running
            │
            └─► Loop:
                    ├─► Get first pending VM (by sequence)
                    │   └─► No more? Set state: done, exit
                    │
                    ├─► Set VM state: running
                    │
                    ├─► Execute VM work units
                    │   ├─► Success: Set VM state: completed
                    │   ├─► VM error: Set VM state: error, continue
                    │   └─► Context canceled: Set state: canceled, exit
                    │
                    └─► Continue loop
```

---

### 3. InspectionStore (`internal/store/inspection.go`)

Manages persistence of VM inspection states using DuckDB.

#### Database Schema

```sql
-- Sequence for FIFO ordering
CREATE SEQUENCE IF NOT EXISTS vm_inspection_status_seq START 1;

-- VM Inspection status table
CREATE TABLE IF NOT EXISTS vm_inspection_status (
    "VM ID" VARCHAR PRIMARY KEY,
    status VARCHAR NOT NULL,
    error VARCHAR,
    sequence INTEGER DEFAULT nextval('vm_inspection_status_seq'),
    FOREIGN KEY ("VM ID") REFERENCES vinfo("VM ID")
);
```

#### Store Interface

```go
type InspectionStore struct {
    db QueryInterceptor
}

// Single VM operations
func (s *InspectionStore) Get(ctx context.Context, vmID string) (*models.InspectionStatus, error)

// Bulk operations
func (s *InspectionStore) List(ctx context.Context, filter *filters.InspectionQueryFilter) (map[string]models.InspectionStatus, error)
func (s *InspectionStore) Add(ctx context.Context, vmIDs []string, status models.InspectionState) error
func (s *InspectionStore) Update(ctx context.Context, filter *filters.InspectionUpdateFilter, status models.InspectionStatus) error
func (s *InspectionStore) DeleteAll(ctx context.Context) error

// Queue operations
func (s *InspectionStore) First(ctx context.Context) (string, error)  // Returns first pending VM by sequence
```

#### Filter System

Two filter types support flexible queries:

```go
// Query filter (SELECT operations)
type InspectionQueryFilter struct {
    filters []InspectionFilterFunc
}
func (f *InspectionQueryFilter) ByVmIDs(vmIDs ...string) *InspectionQueryFilter
func (f *InspectionQueryFilter) ByStatus(statuses ...models.InspectionState) *InspectionQueryFilter
func (f *InspectionQueryFilter) Limit(limit int) *InspectionQueryFilter
func (f *InspectionQueryFilter) OrderBySequence() *InspectionQueryFilter

// Update filter (UPDATE operations)
type InspectionUpdateFilter struct {
    filters []UpdateFilterFunc
}
func (f *InspectionUpdateFilter) ByVmIDs(vmIDs ...string) *InspectionUpdateFilter
func (f *InspectionUpdateFilter) ByStatus(statuses ...models.InspectionState) *InspectionUpdateFilter
```

---

### 4. Work Builder (`pkg/vmware/work_builder.go`)

Builds work units for the inspector workflow.

#### InsWorkBuilder Structure

```go
type InsWorkBuilder struct {
    operator VMOperator           // VMware operations interface
    creds    *models.Credentials  // vSphere credentials
}

func NewInspectorWorkBuilder(cred *models.Credentials) *InsWorkBuilder

func (b *InsWorkBuilder) Init() models.InspectorWorkUnit      // Connection work
func (b *InsWorkBuilder) Build(moid string) []models.InspectorWorkUnit  // Per-VM work
```

#### Work Units

**1. Init Work Unit**
- Creates vSphere client connection
- Sets state to `started`
- Stores VMOperator for subsequent operations

**2. VM Inspection Work Unit**
- Creates snapshot: `assisted-migration-deep-inspector`
- Runs inspection logic (currently placeholder with sleep)
- Removes snapshot with consolidation
- Sets state to `running`

#### Snapshot Management

```go
type CreateSnapshotRequest struct {
    VmMoid       string
    SnapshotName string  // "assisted-migration-deep-inspector"
    Description  string
    Memory       bool    // false - no memory snapshot
    Quiesce      bool    // false - no filesystem quiesce
}

type RemoveSnapshotRequest struct {
    VmMoid       string
    SnapshotName string
    Consolidate  bool    // true - consolidate disks after removal
}
```

---

### 5. REST API (`api/v1/openapi.yaml`)

#### Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/vms/inspector` | `GetInspectorStatus` | Get overall inspector status |
| `POST` | `/vms/inspector` | `StartInspection` | Start inspection for VMs |
| `PATCH` | `/vms/inspector` | `AddVMsToInspection` | Add more VMs to queue |
| `DELETE` | `/vms/inspector` | `StopInspection` | Stop inspector entirely |
| `GET` | `/vms/{id}/inspector` | `GetVMInspectionStatus` | Get single VM inspection status |
| `DELETE` | `/vms/{id}/inspector` | `RemoveVMFromInspection` | Cancel single VM inspection |

#### Request/Response Types

**InspectorStartRequest**
```yaml
InspectorStartRequest:
  type: object
  required:
    - VcenterCredentials
    - vmIds
  properties:
    VcenterCredentials:
      $ref: '#/components/schemas/VcenterCredentials'
    vmIds:
      $ref: '#/components/schemas/VMIdArray'
```

**InspectorStatus**
```yaml
InspectorStatus:
  type: object
  required:
    - state
  properties:
    state:
      type: string
      enum: [ready, started, running, canceling, canceled, done, error]
    error:
      type: string
```

**InspectionStatus** (per-VM)
```yaml
InspectionStatus:
  type: object
  required:
    - state
  properties:
    state:
      type: string
      enum: [pending, running, completed, canceled, error, not_found]
    error:
      type: string
    results:
      type: object
```

#### HTTP Status Codes

| Code | Scenario |
|------|----------|
| `200` | Success (GET, DELETE) |
| `202` | Accepted (POST, PATCH - async operations) |
| `400` | Invalid request body or parameters |
| `404` | Inspector not running (for operations requiring active inspector) |
| `423 Locked` | Inspector already in progress |
| `500` | Internal server error |

---

### 6. Handlers (`internal/handlers/vms.go`)

#### Handler Methods

```go
// Inspector management
func (h *Handler) GetInspectorStatus(c *gin.Context)
func (h *Handler) StartInspection(c *gin.Context)
func (h *Handler) AddVMsToInspection(c *gin.Context)
func (h *Handler) StopInspection(c *gin.Context)

// Per-VM inspection
func (h *Handler) GetVMInspectionStatus(c *gin.Context, id string)
func (h *Handler) RemoveVMFromInspection(c *gin.Context, id string)
```

#### Validation Rules

**StartInspection**
- URL, username, password required
- At least one VM ID required
- Inspector must not be busy

**AddVMsToInspection**
- At least one VM ID required
- Inspector must be running
- Inspector must not be canceling

**RemoveVMFromInspection**
- Inspector must be running
- Inspector must not be canceling

---

### 7. Error Handling (`pkg/errors/errors.go`)

**InspectorWorkError**
```go
type InspectorWorkError struct {
    msg string
}

func NewInspectorWorkError(format string, args ...any) error
```

Used to wrap VM-level errors. When a work unit returns this error type:
- The VM is marked with error state
- The inspector continues to the next VM
- The overall inspector does NOT fail

For other error types (e.g., context cancellation, store errors):
- The inspector stops
- State transitions to error or canceled

---

## Data Flow Diagrams

### Starting Inspection

```
Client                Handler               InspectorService          Store            vSphere
  │                      │                        │                     │                  │
  │  POST /vms/inspector │                        │                     │                  │
  │─────────────────────►│                        │                     │                  │
  │                      │                        │                     │                  │
  │                      │  Validate request      │                     │                  │
  │                      │  Check IsBusy()        │                     │                  │
  │                      │───────────────────────►│                     │                  │
  │                      │◄───────────────────────│                     │                  │
  │                      │                        │                     │                  │
  │                      │  Start(vmIDs, creds)   │                     │                  │
  │                      │───────────────────────►│                     │                  │
  │                      │                        │  Init work          │                  │
  │                      │                        │──────────────────────────────────────►│
  │                      │                        │◄─────────────────────────────────────│
  │                      │                        │                     │                  │
  │                      │                        │  DeleteAll()        │                  │
  │                      │                        │────────────────────►│                  │
  │                      │                        │◄────────────────────│                  │
  │                      │                        │                     │                  │
  │                      │                        │  Add(vmIDs,pending) │                  │
  │                      │                        │────────────────────►│                  │
  │                      │                        │◄────────────────────│                  │
  │                      │                        │                     │                  │
  │                      │                        │  spawn run()        │                  │
  │                      │◄───────────────────────│  goroutine          │                  │
  │  202 Accepted        │                        │                     │                  │
  │◄─────────────────────│                        │                     │                  │
  │                      │                        │                     │                  │
```

### VM Processing Loop

```
InspectorService                    Store                   Scheduler               vSphere
       │                              │                         │                      │
       │  First() - get next pending  │                         │                      │
       │─────────────────────────────►│                         │                      │
       │◄─────────────────────────────│ vm-123                  │                      │
       │                              │                         │                      │
       │  Update(vm-123, running)     │                         │                      │
       │─────────────────────────────►│                         │                      │
       │◄─────────────────────────────│                         │                      │
       │                              │                         │                      │
       │  AddWork(snapshot_create)    │                         │                      │
       │─────────────────────────────────────────────────────►│                      │
       │                              │                         │  CreateSnapshot      │
       │                              │                         │─────────────────────►│
       │                              │                         │◄─────────────────────│
       │◄────────────────────────────────────────────────────│  result               │
       │                              │                         │                      │
       │  AddWork(inspect + cleanup)  │                         │                      │
       │─────────────────────────────────────────────────────►│                      │
       │                              │                         │  [Inspection]        │
       │                              │                         │  RemoveSnapshot      │
       │                              │                         │─────────────────────►│
       │                              │                         │◄─────────────────────│
       │◄────────────────────────────────────────────────────│  result               │
       │                              │                         │                      │
       │  Update(vm-123, completed)   │                         │                      │
       │─────────────────────────────►│                         │                      │
       │◄─────────────────────────────│                         │                      │
       │                              │                         │                      │
       │  [Loop to next VM]           │                         │                      │
       │                              │                         │                      │
```

---

## Testing Strategy

### Unit Tests (`internal/services/inspector_test.go`)

The implementation includes comprehensive Ginkgo/Gomega tests:

#### InspectorService Tests
- Initial state verification
- `IsBusy()` status checks
- Adding VMs to queue
- Getting VM status
- Canceling specific VMs
- Canceling all pending VMs
- Start flow:
  - Single VM success
  - Multiple VM success
  - Processing order verification
  - Init failure handling
  - VM-level error handling (continue to next)
  - Clearing previous data on new start
  - Busy state during execution
- Cancel flow:
  - Stop and cancel all pending

#### InspectionStore Tests
- Add operations (including duplicate handling)
- Get operations (existing and non-existing)
- List with filters (status, VM IDs, limit)
- First (queue ordering by sequence)
- Update operations
- DeleteAll
- Processing order maintenance

### Mock Work Builder

```go
type mockInspectorWorkBuilder struct {
    initErr   error
    vmWorkErr map[string]error  // per-VM errors
    workDelay time.Duration
    inspected []string
    mu        sync.Mutex
}
```

Supports:
- Configurable init errors
- Per-VM error injection
- Work delay simulation
- Inspection tracking

---

## Configuration & Wiring

### Service Initialization (`cmd/run.go`)

```go
// Create inspector service
inspectorSrv := services.NewInspectorService(sched, s)

// Initialize handlers with all services
h := handlers.New(consoleSrv, collectorSrv, inventorySrv, vmSrv, inspectorSrv)
```

### Dependencies
- **Scheduler**: Shared with Collector service for work execution
- **Store**: Access to inspection table and VM info table (foreign key)
- **VMware package**: vSphere client and VM operations

---

## Future Enhancements

The current implementation includes placeholders for:

1. **Inspection Logic** (`work_builder.go:84`)
   ```go
   // Todo: add the inspection logic here
   time.Sleep(180 * time.Second)
   ```
   Future: Disk analysis, OS detection, application discovery

2. **Cleanup Handling** (`inspector.go:188`)
   ```go
   // Todo: handle the context done case. we may want to run some cleanup tasks
   ```
   Future: Ensure snapshots are removed on cancellation

3. **Request Validation** (`vms.go:225`)
   ```go
   // Todo: validate using the openapi spec. do the same for the collector
   ```
   Future: OpenAPI-based request validation

4. **Inspection Results**
   The `InspectionStatus` schema includes a `results` field (currently unused):
   ```yaml
   results:
     type: object
     description: Inspection results
   ```

---

## API Examples

### Start Inspection

```bash
curl -X POST http://localhost:8080/api/v1/vms/inspector \
  -H "Content-Type: application/json" \
  -d '{
    "VcenterCredentials": {
      "url": "https://vcenter.example.com",
      "username": "admin@vsphere.local",
      "password": "secret"
    },
    "vmIds": ["vm-1001", "vm-1002", "vm-1003"]
  }'

# Response: 202 Accepted
{
  "state": "started"
}
```

### Check Inspector Status

```bash
curl http://localhost:8080/api/v1/vms/inspector

# Response: 200 OK
{
  "state": "running"
}
```

### Check VM Inspection Status

```bash
curl http://localhost:8080/api/v1/vms/vm-1001/inspector

# Response: 200 OK
{
  "state": "completed"
}
```

### Add More VMs to Queue

```bash
curl -X PATCH http://localhost:8080/api/v1/vms/inspector \
  -H "Content-Type: application/json" \
  -d '["vm-1004", "vm-1005"]'

# Response: 202 Accepted
{
  "state": "running"
}
```

### Remove VM from Queue

```bash
curl -X DELETE http://localhost:8080/api/v1/vms/vm-1003/inspector

# Response: 200 OK
{
  "state": "canceled"
}
```

### Stop Inspector

```bash
curl -X DELETE http://localhost:8080/api/v1/vms/inspector

# Response: 200 OK
{
  "state": "canceled"
}
```

---

## Glossary

| Term | Definition |
|------|------------|
| **MOID** | Managed Object ID - VMware's unique identifier for VMs (e.g., "vm-1001") |
| **Work Unit** | A single atomic operation in the inspection workflow |
| **Work Builder** | Factory that creates work units for the inspector |
| **Sequence** | Auto-incrementing value ensuring FIFO processing order |
| **Snapshot** | Point-in-time copy of VM state for safe inspection |

---

## References

- Existing Collector service pattern: `internal/services/collector.go`
- Scheduler implementation: `pkg/scheduler/scheduler.go`
- VMware client: `pkg/vmware/client.go`
- OpenAPI specification: `api/v1/openapi.yaml`

