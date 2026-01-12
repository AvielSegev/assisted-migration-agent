package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	v1 "github.com/kubev2v/assisted-migration-agent/api/v1"
	"github.com/kubev2v/assisted-migration-agent/internal/services"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// GetVMs returns the list of VMs with filtering and pagination
// (GET /vms)
func (h *Handler) GetVMs(c *gin.Context, params v1.GetVMsParams) {
	// Parse pagination
	page := 1
	if params.Page != nil && *params.Page > 0 {
		page = *params.Page
	}
	pageSize := defaultPageSize
	if params.PageSize != nil && *params.PageSize > 0 {
		pageSize = *params.PageSize
		if pageSize > maxPageSize {
			pageSize = maxPageSize
		}
	}

	// Build service params
	svcParams := services.VMListParams{
		Limit:  uint64(pageSize),
		Offset: uint64((page - 1) * pageSize),
	}

	if params.Datacenters != nil {
		svcParams.Datacenters = *params.Datacenters
	}
	if params.Clusters != nil {
		svcParams.Clusters = *params.Clusters
	}
	if params.Status != nil {
		svcParams.Statuses = *params.Status
	}
	if params.Issues != nil {
		svcParams.Issues = *params.Issues
	}
	if params.Disksize != nil {
		svcParams.DiskRanges = v1.ParseDiskSizeRanges(*params.Disksize)
	}
	if params.Memorysize != nil {
		svcParams.MemRanges = v1.ParseMemorySizeRanges(*params.Memorysize)
	}

	result, err := h.vmSrv.List(c.Request.Context(), svcParams)
	if err != nil {
		zap.S().Named("vm_handler").Errorw("failed to list VMs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list VMs"})
		return
	}

	// Calculate page count
	pageCount := (result.Total + pageSize - 1) / pageSize
	if pageCount == 0 {
		pageCount = 1
	}

	// Map to API response
	apiVMs := make([]v1.VM, 0, len(result.VMs))
	for _, vm := range result.VMs {
		apiVMs = append(apiVMs, v1.NewVMFromModel(vm))
	}

	c.JSON(http.StatusOK, v1.VMListResponse{
		Page:      page,
		PageCount: pageCount,
		Total:     result.Total,
		Vms:       apiVMs,
	})
}

// GetVMInspectionStatus returns the inspection status for a specific VM
// (GET /vms/{id}/inspector)
func (h *Handler) GetVMInspectionStatus(c *gin.Context, id int) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not yet implemented"})
}

// GetInspectorStatus returns the inspector status
// (GET /vms/inspector)
func (h *Handler) GetInspectorStatus(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not yet implemented"})
}

// StartInspection starts inspection for VMs
// (POST /vms/inspector)
func (h *Handler) StartInspection(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not yet implemented"})
}

// AddVMsToInspection adds more VMs to inspection queue
// (PATCH /vms/inspector)
func (h *Handler) AddVMsToInspection(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not yet implemented"})
}

// RemoveVMsFromInspection removes VMs from inspection queue or stops inspector entirely
// (DELETE /vms/inspector)
func (h *Handler) RemoveVMsFromInspection(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not yet implemented"})
}
