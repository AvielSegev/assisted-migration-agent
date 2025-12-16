package handlers

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	v1 "github.com/kubev2v/assisted-migration-agent/api/v1"
	"github.com/kubev2v/assisted-migration-agent/internal/models"
)

// GetCollectorStatus returns the collector status
// (GET /collector)
func (h *Handler) GetCollectorStatus(c *gin.Context) {
	// Check if credentials exist to determine status
	exists, err := h.collector.HasCredentials(c.Request.Context())
	if err != nil {
		zap.S().Errorw("failed to check credentials existence", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get collector status"})
		return
	}

	// TODO: Get actual status from collector service based on credentials existence
	// For now, always return idle until collector state machine is implemented
	_ = exists

	c.JSON(http.StatusOK, v1.CollectorStatus{
		Status: v1.CollectorStatusStatusIdle,
	})
}

// StartCollector starts inventory collection
// (POST /collector)
func (h *Handler) StartCollector(c *gin.Context) {
	var req v1.CollectorStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate required fields
	if req.Url == "" || req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url, username, and password are required"})
		return
	}

	// Validate URL format
	parsedURL, err := url.Parse(req.Url)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url format"})
		return
	}

	// Save credentials via collector service
	creds := &models.Credentials{
		URL:      req.Url,
		Username: req.Username,
		Password: req.Password,
	}
	if err := h.collector.SaveCredentials(c.Request.Context(), creds); err != nil {
		zap.S().Errorw("failed to save credentials", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save credentials"})
		return
	}

	// TODO: Verify credentials with vCenter before starting collection
	// TODO: Start async collection job via scheduler

	c.JSON(http.StatusAccepted, v1.CollectorStatus{
		Status: v1.CollectorStatusStatusConnecting,
	})
}

// GetInventory returns the collected inventory
// (GET /collector/inventory)
func (h *Handler) GetInventory(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"msg": "not implemented"})
}

// StopCollector stops the collection
// (DELETE /collector)
func (h *Handler) StopCollector(c *gin.Context) {
	// Delete credentials via collector service
	if err := h.collector.DeleteCredentials(c.Request.Context()); err != nil {
		zap.S().Errorw("failed to delete credentials", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete credentials"})
		return
	}

	// TODO: Cancel any running collection job

	c.JSON(http.StatusOK, v1.CollectorStatus{
		Status: v1.CollectorStatusStatusIdle,
	})
}
