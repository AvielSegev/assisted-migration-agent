package services_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubev2v/assisted-migration-agent/internal/models"
	"github.com/kubev2v/assisted-migration-agent/internal/services"
	"github.com/kubev2v/assisted-migration-agent/internal/store"
	"github.com/kubev2v/assisted-migration-agent/internal/store/migrations"
	srvErrors "github.com/kubev2v/assisted-migration-agent/pkg/errors"
)

// getVCenterCredentials returns test credentials for vCenter.
// vcsim accepts any username/password, but we use standard test values.
func getVCenterCredentials() *models.Credentials {
	return &models.Credentials{
		URL:      "https://localhost:8989/sdk",
		Username: "user",
		Password: "pass",
	}
}

// mockInspectorBuilder provides a configurable InspectionWorkBuilder for tests.
type mockInspectorBuilder struct {
	delay     time.Duration
	vmErrors  map[string]error
	inspected []string
	mu        sync.Mutex
}

func (m *mockInspectorBuilder) withWorkDelay(d time.Duration) *mockInspectorBuilder {
	m.delay = d
	return m
}

func (m *mockInspectorBuilder) withVmError(vmID string, err error) *mockInspectorBuilder {
	if m.vmErrors == nil {
		m.vmErrors = make(map[string]error)
	}
	m.vmErrors[vmID] = err
	return m
}

func (m *mockInspectorBuilder) getInspectedVMs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.inspected...)
}

func (m *mockInspectorBuilder) builder() services.InspectionWorkBuilder {
	return func(id string) []models.WorkUnit[models.InspectionStatus, models.InspectionResult] {
		return []models.WorkUnit[models.InspectionStatus, models.InspectionResult]{
			{
				Status: func() models.InspectionStatus {
					return models.InspectionStatus{State: models.InspectionStateRunning}
				},
				Work: func(ctx context.Context, result models.InspectionResult) (models.InspectionResult, error) {
					if m.delay > 0 {
						select {
						case <-time.After(m.delay):
						case <-ctx.Done():
							return result, ctx.Err()
						}
					}
					if err, ok := m.vmErrors[id]; ok && err != nil {
						return result, err
					}
					m.mu.Lock()
					m.inspected = append(m.inspected, id)
					m.mu.Unlock()
					return result, nil
				},
			},
		}
	}
}

func newMockInspectorWorkBuilder() *mockInspectorBuilder {
	return &mockInspectorBuilder{}
}

var _ = Describe("InspectorService", func() {
	var (
		ctx context.Context
		db  *sql.DB
		srv *services.InspectorService
	)

	// Helper to insert test VMs into vinfo table
	insertVM := func(id, name string) {
		_, err := db.ExecContext(ctx, `
			INSERT INTO vinfo ("VM ID", "VM", "Powerstate", "Cluster", "Memory")
			VALUES (?, ?, 'poweredOn', 'cluster-a', 4096)
		`, id, name)
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		ctx = context.Background()

		var err error
		db, err = store.NewDB(nil, ":memory:")
		Expect(err).NotTo(HaveOccurred())

		err = migrations.Run(ctx, db)
		Expect(err).NotTo(HaveOccurred())

		// Insert test VMs into vinfo (required for foreign key constraint)
		insertVM("vm-1", "test-vm-1")
		insertVM("vm-2", "test-vm-2")
		insertVM("vm-3", "test-vm-3")

		srv = services.NewInspectorService()
	})

	AfterEach(func() {
		if srv != nil {
			_ = srv.Stop(ctx)
		}
		if db != nil {
			_ = db.Close()
		}
	})

	Describe("GetStatus", func() {
		It("should return ready state initially", func() {
			status := srv.GetStatus()
			Expect(status.State).To(Equal(models.InspectorStateReady))
		})
	})

	Describe("IsBusy", func() {
		It("should return false when in ready state", func() {
			Expect(srv.IsBusy()).To(BeFalse())
		})
	})

	Describe("Add VMs to inspectionSvc queue", func() {

		Context("when inspector is not started", func() {
			It("should return InspectorNotRunningError when trying to add VMs", func() {
				err := srv.Add([]string{"vm-1", "vm-2"})
				Expect(err).To(HaveOccurred())

				var notRunningErr *srvErrors.InspectorNotRunningError
				Expect(errors.As(err, &notRunningErr)).To(BeTrue())
			})
		})

		Context("when inspector is running", func() {
			BeforeEach(func() {
				// Insert an initial VM for starting the inspector
				insertVM("vm-0", "test-vm-0")

				// Use a mock builder with delay to keep inspector running
				builder := newMockInspectorWorkBuilder().withWorkDelay(1 * time.Second)
				srv = services.NewInspectorService().WithWorkUnitsBuilder(builder.builder())

				// Start inspector with vm-0 (will stay running due to delay)
				err := srv.Start(ctx, []string{"vm-0"}, getVCenterCredentials())
				Expect(err).NotTo(HaveOccurred())

				// Wait for inspector to be in running state
				Eventually(func() models.InspectorState {
					return srv.GetStatus().State
				}).Should(Equal(models.InspectorStateRunning))
			})

			It("should add VMs to inspectionSvc table with pending status", func() {
				err := srv.Add([]string{"vm-1", "vm-2", "vm-3"})
				Expect(err).NotTo(HaveOccurred())

				// Verify added VMs are in pipelines (pending or running)
				for _, vmID := range []string{"vm-1", "vm-2", "vm-3"} {
					status, err := srv.GetVmStatus(ctx, vmID)
					Expect(err).NotTo(HaveOccurred())
					Expect(status.State).To(Or(
						Equal(models.InspectionStatePending),
						Equal(models.InspectionStateRunning),
					))
				}
			})

			It("should not duplicate VMs when adding same VM twice", func() {
				err := srv.Add([]string{"vm-1", "vm-2"})
				Expect(err).NotTo(HaveOccurred())

				err = srv.Add([]string{"vm-2", "vm-3"})
				Expect(err).NotTo(HaveOccurred())

				// Should have vm-0 (from Start) + vm-1, vm-2, vm-3 (from Add) = 4 total in pipelines
				for _, vmID := range []string{"vm-0", "vm-1", "vm-2", "vm-3"} {
					status, err := srv.GetVmStatus(ctx, vmID)
					Expect(err).NotTo(HaveOccurred())
					Expect(status.State).NotTo(Equal(models.InspectionStateNotFound))
				}
			})

			It("should return error for empty VM list", func() {
				err := srv.Add([]string{})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetVmStatus", func() {
		It("should return NotFound state for non-existent VM", func() {
			status, err := srv.GetVmStatus(ctx, "non-existent")
			Expect(err).NotTo(HaveOccurred())
			Expect(status.State).To(Equal(models.InspectionStateNotFound))
		})

		It("should return VM inspectionSvc status after adding", func() {
			// Insert an initial VM and start inspector
			insertVM("vm-0", "test-vm-0")

			// Use a mock builder with delay to keep inspector running
			builder := newMockInspectorWorkBuilder().withWorkDelay(1 * time.Second)
			srv = services.NewInspectorService().WithWorkUnitsBuilder(builder.builder())

			err := srv.Start(ctx, []string{"vm-0"}, getVCenterCredentials())
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() models.InspectorState {
				return srv.GetStatus().State
			}).Should(Equal(models.InspectorStateRunning))

			err = srv.Add([]string{"vm-1"})
			Expect(err).NotTo(HaveOccurred())

			status, err := srv.GetVmStatus(ctx, "vm-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(status.State).To(Or(
				Equal(models.InspectionStatePending),
				Equal(models.InspectionStateRunning),
			))
		})
	})

	Describe("CancelVmsInspection", func() {

		Context("when inspector is not started", func() {
			It("should return InspectorNotRunningError when trying to cancel VMs", func() {
				err := srv.CancelVmsInspection(ctx, "vm-1", "vm-2")
				Expect(err).To(HaveOccurred())

				var notRunningErr *srvErrors.InspectorNotRunningError
				Expect(errors.As(err, &notRunningErr)).To(BeTrue())
			})

			It("should return InspectorNotRunningError when trying to cancel all VMs", func() {
				err := srv.CancelVmsInspection(ctx)
				Expect(err).To(HaveOccurred())

				var notRunningErr *srvErrors.InspectorNotRunningError
				Expect(errors.As(err, &notRunningErr)).To(BeTrue())
			})
		})

		Context("when inspector is running", func() {
			BeforeEach(func() {
				// Insert an initial VM for starting the inspector
				insertVM("vm-0", "test-vm-0")

				// Use a mock builder with delay to keep inspector running
				builder := newMockInspectorWorkBuilder().withWorkDelay(1 * time.Second)
				srv = services.NewInspectorService().WithWorkUnitsBuilder(builder.builder())

				// Start inspector with vm-0 (will stay running due to delay)
				err := srv.Start(ctx, []string{"vm-0"}, getVCenterCredentials())
				Expect(err).NotTo(HaveOccurred())

				// Wait for inspector to be in running state
				Eventually(func() models.InspectorState {
					return srv.GetStatus().State
				}).Should(Equal(models.InspectorStateRunning))

				// Add VMs to the inspectionSvc queue
				err = srv.Add([]string{"vm-1", "vm-2", "vm-3"})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should cancel specific pending VMs", func() {
				err := srv.CancelVmsInspection(ctx, "vm-2")
				Expect(err).NotTo(HaveOccurred())

				// Check vm-2 status is canceled
				status, err := srv.GetVmStatus(ctx, "vm-2")
				Expect(err).NotTo(HaveOccurred())
				Expect(status.State).To(Equal(models.InspectionStateCanceled))

				// Other VMs should still be pending or running
				status1, err := srv.GetVmStatus(ctx, "vm-1")
				Expect(err).NotTo(HaveOccurred())
				Expect(status1.State).To(Or(
					Equal(models.InspectionStatePending),
					Equal(models.InspectionStateRunning),
				))

				status3, err := srv.GetVmStatus(ctx, "vm-3")
				Expect(err).NotTo(HaveOccurred())
				Expect(status3.State).To(Or(
					Equal(models.InspectionStatePending),
					Equal(models.InspectionStateRunning),
				))
			})

			It("should cancel multiple specific VMs", func() {
				err := srv.CancelVmsInspection(ctx, "vm-1", "vm-3")
				Expect(err).NotTo(HaveOccurred())

				// Check vm-1 and vm-3 are canceled
				status1, err := srv.GetVmStatus(ctx, "vm-1")
				Expect(err).NotTo(HaveOccurred())
				Expect(status1.State).To(Equal(models.InspectionStateCanceled))

				status3, err := srv.GetVmStatus(ctx, "vm-3")
				Expect(err).NotTo(HaveOccurred())
				Expect(status3.State).To(Equal(models.InspectionStateCanceled))

				// vm-2 should still be pending or running
				status2, err := srv.GetVmStatus(ctx, "vm-2")
				Expect(err).NotTo(HaveOccurred())
				Expect(status2.State).To(Or(
					Equal(models.InspectionStatePending),
					Equal(models.InspectionStateRunning),
				))
			})

			It("should cancel all pending VMs when no specific IDs provided", func() {
				err := srv.CancelVmsInspection(ctx)
				Expect(err).NotTo(HaveOccurred())

				// All VMs in pipelines (vm-0, vm-1, vm-2, vm-3) should be canceled
				for _, vmID := range []string{"vm-0", "vm-1", "vm-2", "vm-3"} {
					status, err := srv.GetVmStatus(ctx, vmID)
					Expect(err).NotTo(HaveOccurred())
					Expect(status.State).To(Equal(models.InspectionStateCanceled))
				}
			})
		})
	})

	Describe("Start", func() {
		It("should complete inspectionSvc successfully for single VM", func() {
			builder := newMockInspectorWorkBuilder()
			srv = services.NewInspectorService().WithWorkUnitsBuilder(builder.builder())

			err := srv.Start(ctx, []string{"vm-1"}, getVCenterCredentials())
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() models.InspectorState {
				return srv.GetStatus().State
			}).Should(Equal(models.InspectorStateCompleted))

			// Verify VM was inspected
			Expect(builder.getInspectedVMs()).To(ContainElement("vm-1"))

			// Verify VM status is completed
			status, err := srv.GetVmStatus(ctx, "vm-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(status.State).To(Equal(models.InspectionStateCompleted))
		})

		It("should complete inspectionSvc successfully for multiple VMs", func() {
			builder := newMockInspectorWorkBuilder()
			srv = services.NewInspectorService().WithWorkUnitsBuilder(builder.builder())

			err := srv.Start(ctx, []string{"vm-1", "vm-2", "vm-3"}, getVCenterCredentials())
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() models.InspectorState {
				return srv.GetStatus().State
			}).Should(Equal(models.InspectorStateCompleted))

			// Verify all VMs were inspected
			inspected := builder.getInspectedVMs()
			Expect(inspected).To(HaveLen(3))
			Expect(inspected).To(ContainElements("vm-1", "vm-2", "vm-3"))
		})

		It("should process VMs in sequence order", func() {
			builder := newMockInspectorWorkBuilder()
			srv = services.NewInspectorService().WithWorkUnitsBuilder(builder.builder())

			err := srv.Start(ctx, []string{"vm-1", "vm-2", "vm-3"}, getVCenterCredentials())
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() models.InspectorState {
				return srv.GetStatus().State
			}).Should(Equal(models.InspectorStateCompleted))

			// VMs should be processed in order they were added
			Expect(builder.getInspectedVMs()).To(Equal([]string{"vm-1", "vm-2", "vm-3"}))
		})

		It("should return error for invalid cred", func() {
			// Use invalid credentials to trigger connection error
			invalidCreds := &models.Credentials{
				URL:      "https://invalid-host:8989/sdk",
				Username: "invalid",
				Password: "invalid",
			}

			err := srv.Start(ctx, []string{"vm-1"}, invalidCreds)
			Expect(err).To(HaveOccurred())
			errMsg := err.Error()
			Expect(errMsg).To(Or(
				ContainSubstring("connection refused"),
				ContainSubstring("no such host"),
				ContainSubstring("timeout"),
				ContainSubstring("connection"),
				ContainSubstring("failed to connect"),
				ContainSubstring("dial tcp"),
			))

			status := srv.GetStatus()
			Expect(status.State).To(Equal(models.InspectorStateError))
			Expect(status.Error).NotTo(BeNil())
		})

		It("should mark VM as error when inspectionSvc fails and continue with next VM", func() {
			builder := newMockInspectorWorkBuilder().withVmError("vm-1", errors.New("inspectionSvc failed"))
			srv = services.NewInspectorService().WithWorkUnitsBuilder(builder.builder())

			err := srv.Start(ctx, []string{"vm-1", "vm-2"}, getVCenterCredentials())
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() models.InspectorState {
				return srv.GetStatus().State
			}).Should(Equal(models.InspectorStateCompleted))

			// Check vm-1 status is error
			status1, err := srv.GetVmStatus(ctx, "vm-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(status1.State).To(Equal(models.InspectionStateError))
			Expect(status1.Error).NotTo(BeNil())

			// Check vm-2 status is completed (should continue after vm-1 error)
			status2, err := srv.GetVmStatus(ctx, "vm-2")
			Expect(err).NotTo(HaveOccurred())
			Expect(status2.State).To(Equal(models.InspectionStateCompleted))
		})

		It("should clear previous inspectionSvc data on new start", func() {
			builder := newMockInspectorWorkBuilder()
			srv = services.NewInspectorService().WithWorkUnitsBuilder(builder.builder())

			// First run
			err := srv.Start(ctx, []string{"vm-1"}, getVCenterCredentials())
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() models.InspectorState {
				return srv.GetStatus().State
			}).Should(Equal(models.InspectorStateCompleted))

			err = srv.Start(ctx, []string{"vm-2", "vm-3"}, getVCenterCredentials())
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() models.InspectorState {
				return srv.GetStatus().State
			}).Should(Equal(models.InspectorStateCompleted))

			// Should only have vm-2 and vm-3 in pipelines; vm-1 from first run is gone
			status1, _ := srv.GetVmStatus(ctx, "vm-1")
			Expect(status1.State).To(Equal(models.InspectionStateNotFound))
			status2, err := srv.GetVmStatus(ctx, "vm-2")
			Expect(err).NotTo(HaveOccurred())
			Expect(status2.State).To(Equal(models.InspectionStateCompleted))
			status3, err := srv.GetVmStatus(ctx, "vm-3")
			Expect(err).NotTo(HaveOccurred())
			Expect(status3.State).To(Equal(models.InspectionStateCompleted))
		})

		It("should be busy while running", func() {
			builder := newMockInspectorWorkBuilder().withWorkDelay(100 * time.Millisecond)
			srv = services.NewInspectorService().WithWorkUnitsBuilder(builder.builder())

			err := srv.Start(ctx, []string{"vm-1"}, getVCenterCredentials())
			Expect(err).NotTo(HaveOccurred())

			// Should be busy while running
			Eventually(func() bool {
				return srv.IsBusy()
			}).Should(BeTrue())

			// Wait for completion
			Eventually(func() models.InspectorState {
				return srv.GetStatus().State
			}).Should(Equal(models.InspectorStateCompleted))

			// Should not be busy after completion
			Expect(srv.IsBusy()).To(BeFalse())
		})
	})

	Describe("CancelInspector", func() {
		It("should stop inspector and cancel all pending VMs", func() {
			builder := newMockInspectorWorkBuilder().withWorkDelay(1 * time.Second)
			srv = services.NewInspectorService().WithWorkUnitsBuilder(builder.builder())

			err := srv.Start(ctx, []string{"vm-1", "vm-2", "vm-3"}, getVCenterCredentials())
			Expect(err).NotTo(HaveOccurred())

			// Wait for running state
			Eventually(func() models.InspectorState {
				return srv.GetStatus().State
			}).Should(Equal(models.InspectorStateRunning))

			// Cancel inspector
			err = srv.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Inspector should be in canceled state
			status := srv.GetStatus()
			Expect(status.State).To(Equal(models.InspectorStateCanceled))

			// Should not be busy
			Expect(srv.IsBusy()).To(BeFalse())
		})
	})

})
