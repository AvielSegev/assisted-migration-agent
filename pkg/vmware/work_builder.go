package vmware

import (
	"context"

	"go.uber.org/zap"

	"github.com/kubev2v/assisted-migration-agent/internal/models"
)

// InsWorkBuilder builds a sequence of WorkUnits for the v1 Inspector workflow.
type InsWorkBuilder struct {
	operator VMOperator
}

// NewInspectorWorkBuilder creates a new v1 work builder.
func NewInspectorWorkBuilder(operator VMOperator) *InsWorkBuilder {
	return &InsWorkBuilder{
		operator: operator,
	}
}

// Build creates the sequence of WorkUnits for the Inspector workflow.
func (b *InsWorkBuilder) Build(id string) []models.InspectorWorkUnit {
	return b.vmWork(id)
}

func (b *InsWorkBuilder) vmWork(id string) []models.InspectorWorkUnit {
	var units []models.InspectorWorkUnit

	inspect := models.InspectorWorkUnit{
		Work: func() func(ctx context.Context) (any, error) {
			return func(ctx context.Context) (any, error) {
				zap.S().Named("inspector_service").Info("validate privileges on VM")

				if err := b.operator.ValidatePrivileges(ctx, id, models.RequiredPrivileges); err != nil {
					zap.S().Named("inspector_service").Errorw("validation failed", "error", err)
					return nil, err
				}

				zap.S().Named("inspector_service").Infow("creating VM snapshot", "vmId", id)
				req := CreateSnapshotRequest{
					VmId:         id,
					SnapshotName: models.InspectionSnapshotName,
					Description:  "",
					Memory:       false,
					Quiesce:      false,
				}

				if err := b.operator.CreateSnapshot(ctx, req); err != nil {
					zap.S().Named("inspector_service").Errorw("failed to create VM snapshot", "vmId", id, "error", err)
					return nil, err
				}

				zap.S().Named("inspector_service").Infow("VM snapshot created", "vmId", id)

				// Todo: add the inspection logic here

				removeSnapReq := RemoveSnapshotRequest{
					VmId:         id,
					SnapshotName: models.InspectionSnapshotName,
					Consolidate:  true,
				}

				if err := b.operator.RemoveSnapshot(ctx, removeSnapReq); err != nil {
					zap.S().Named("inspector_service").Errorw("failed to remove VM snapshot", "vmId", id, "error", err)
					return nil, err
				}

				zap.S().Named("inspector_service").Infow("VM snapshot removed", "vmId", id)

				return nil, nil
			}
		},
	}

	units = append(units, inspect)

	return units
}
