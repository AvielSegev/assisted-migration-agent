package vmware

import (
	"context"
	"go.uber.org/zap"

	"github.com/kubev2v/assisted-migration-agent/internal/models"
	"github.com/kubev2v/assisted-migration-agent/internal/store"
)

// InsWorkBuilder builds a sequence of WorkUnits for the v1 Inspector workflow.
type InsWorkBuilder struct {
	operator VMOperator
	store    *store.Store
	creds    *models.Credentials
	vmsMoid  []string
}

// NewInspectorWorkBuilder creates a new v1 work builder.
func NewInspectorWorkBuilder(s *store.Store) *InsWorkBuilder {
	return &InsWorkBuilder{
		store: s,
	}
}

// Build creates the sequence of WorkUnits for the Inspector workflow.
func (b *InsWorkBuilder) Build() models.InspectorFlow {
	return models.InspectorFlow{
		Connect: b.connect(),
		Inspect: b.inspectVms(),
	}
}

func (b *InsWorkBuilder) WithVms(vmsMoid []string) models.InspectorWorkBuilder {
	b.vmsMoid = vmsMoid
	return b
}

func (b *InsWorkBuilder) Reset() {
	b.operator = nil
	b.creds = nil
	b.vmsMoid = nil
}

func (b *InsWorkBuilder) connect() models.InspectorWorkUnit {
	return models.InspectorWorkUnit{
		Status: func() models.InspectorStatus {
			return models.InspectorStatus{State: models.InspectorStateConnecting}
		},
		Work: func() func(ctx context.Context) (any, error) {
			return func(ctx context.Context) (any, error) {
				zap.S().Named("inspector_service").Info("loading vCenter credentials")
				credentialsStore := b.store.Credentials()
				creds, err := credentialsStore.Get(ctx)
				if err != nil {
					zap.S().Named("inspector_service").Errorw("failed to load credentials", "error", err)
					return nil, err
				}
				b.creds = creds

				zap.S().Named("inspector_service").Info("connecting to vSphere")
				c, err := NewVsphereClient(ctx, creds.URL, creds.Username, creds.Password, true)
				if err != nil {
					zap.S().Named("inspector_service").Errorw("failed to connect to vSphere", "error", err)
					return nil, err
				}

				b.operator = NewVMManager(c)
				zap.S().Named("inspector_service").Info("vSphere connection established")

				return nil, nil
			}
		},
	}
}

func (b *InsWorkBuilder) inspectVms() models.VmsWork {
	var work models.VmsWork
	work.VmsInitialStatus = make(map[string]models.InspectionStatus)

	var units []models.VmWorkUnit

	for _, vmMoid := range b.vmsMoid {
		moid := vmMoid // capture loop variable
		units = append(units, models.VmWorkUnit{
			VmMoid: moid,
			Work: models.InspectorWorkUnit{
				Status: func() models.InspectorStatus {
					return models.InspectorStatus{State: models.InspectorStateRunning}
				},
				Work: func() func(ctx context.Context) (any, error) {
					return func(ctx context.Context) (any, error) {
						zap.S().Named("inspector_service").Infow("creating VM snapshot", "vmMoid", moid)
						req := CreateSnapshotRequest{
							VmMoid:       moid,
							SnapshotName: models.InspectionSnapshotName,
							Description:  "",
							Memory:       false,
							Quiesce:      false,
						}

						if err := b.operator.CreateSnapshot(ctx, req); err != nil {
							zap.S().Named("inspector_service").Errorw("failed to create VM snapshot", "vmMoid", moid, "error", err)
							return nil, err
						}

						zap.S().Named("inspector_service").Infow("VM snapshot created", "vmMoid", moid)
						return nil, nil
					}
				},
			},
		})

		work.VmsInitialStatus[moid] = models.InspectionStatus{
			State: models.InspectionStatePending,
		}
	}

	work.Works = units

	return work
}
