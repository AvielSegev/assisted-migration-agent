package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kubev2v/assisted-migration-agent/pkg/e2e/backend"
	"github.com/kubev2v/assisted-migration-agent/pkg/e2e/infra"
	"github.com/kubev2v/assisted-migration-agent/test/e2e-v2/service"

	"github.com/google/uuid"
	"github.com/kubev2v/migration-planner/api/v1alpha1"
	"github.com/onsi/ginkgo/v2"
	gm "github.com/onsi/gomega"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

var _ = ginkgo.Describe("Connected env v2 e2e tests", ginkgo.Ordered, func() {
	var (
		plannerSvc *backend.PlannerSvc
		proxy      *infra.Proxy
		oidcProxy  *infra.Proxy
		obs        *infra.Observer
	)

	ginkgo.BeforeAll(func() {
		ginkgo.GinkgoWriter.Println("Starting postgres...")
		err := infraManager.StartPostgres()
		gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to start postgres")
		time.Sleep(2 * time.Second)

		ginkgo.GinkgoWriter.Println("Starting OIDC server...")
		err = infraManager.StartOIDC(":9090")
		gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to start OIDC server")

		oidcUrl, _ := url.Parse("http://localhost:9090")
		oidcProxy = infra.NewProxy("oidc-proxy", "oidc", oidcUrl, ":8082")

		ginkgo.GinkgoWriter.Println("Starting backend...")
		err = infraManager.StartBackend()
		gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to start backend")

		gm.Eventually(func() error {
			resp, err := http.DefaultClient.Get(cfg.BackendAgentEndpoint + "/health")
			if err != nil {
				return err
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 500 {
				return fmt.Errorf("server error: %d", resp.StatusCode)
			}
			return nil
		}, 30*time.Second, 1*time.Second).Should(gm.BeNil())

		plannerSvc = backend.NewPlannerServiceWithOIDC(cfg.BackendUserEndpoint, infraManager.GenerateToken)

		target, err := url.Parse(cfg.BackendAgentEndpoint)
		gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to parse backend endpoint")

		var requests chan infra.Request
		proxy, requests = infra.NewObservableProxy("agent-proxy", "backend", target, ":8081")
		obs = infra.NewObserver(requests)

		time.Sleep(100 * time.Millisecond)
		ginkgo.GinkgoWriter.Println("Proxy started on :8081")
	})

	ginkgo.AfterAll(func() {
		if proxy != nil {
			proxy.Stop()
		}
		if oidcProxy != nil {
			oidcProxy.Stop()
		}
		if obs != nil {
			obs.Close()
		}
		_ = infraManager.StopBackend()
		_ = infraManager.StopOIDC()
		_ = infraManager.StopPostgres()
	})

	ginkgo.Context("mode at startup", func() {
		var (
			agentSvc *service.AgentSvc
			sourceID openapi_types.UUID
			userSvc  *backend.PlannerSvc
		)

		ginkgo.BeforeEach(func() {
			agentSvc = service.DefaultAgentSvc(cfg.AgentAPIUrl)
			userSvc = plannerSvc.WithAuthUser("admin", "admin", "admin@example.com")

			source, err := userSvc.CreateSource("test-source-" + uuid.NewString()[:8])
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to create source")
			sourceID = source.Id
			ginkgo.GinkgoWriter.Printf("Created source: %s\n", sourceID)
		})

		ginkgo.AfterEach(func() {
			if ginkgo.CurrentSpecReport().Failed() {
				ginkgo.GinkgoWriter.Println("Keeping containers running (test failed)")
				return
			}
			ginkgo.GinkgoWriter.Println("Stopping agent...")
			_ = infraManager.RemoveAgent()

			ginkgo.GinkgoWriter.Println("Deleting source...")
			_ = userSvc.RemoveSource(sourceID)
		})

		// Given an agent configured in connected mode with a valid source ID
		// When the agent starts and registers with the backend
		// Then the source should have the agent attached
		ginkgo.It("should register agent with backend when starting in connected mode", func() {
			agentID := uuid.NewString()
			_, err := infraManager.StartAgent(infra.AgentConfig{
				AgentID:        agentID,
				SourceID:       sourceID.String(),
				Mode:           "connected",
				ConsoleURL:     cfg.BackendAgentEndpoint,
				APIVersion:     "v2",
				UpdateInterval: "1s",
			})
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to start agent")
			ginkgo.GinkgoWriter.Printf("Agent started with ID: %s\n", agentID)

			gm.Eventually(func() error {
				_, err := agentSvc.Status()
				return err
			}, 30*time.Second, 1*time.Second).Should(gm.BeNil())

			time.Sleep(10 * time.Second) // allow agent to register with backend
			source, err := userSvc.GetSource(sourceID)
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to get source")

			ginkgo.GinkgoWriter.Printf("Source agent: %+v\n", source.Agent)
			gm.Expect(source.Agent).ToNot(gm.BeNil(), "expected agent to be attached to source")
		})
	})

	ginkgo.Context("mode switch", func() {
		var (
			agentSvc *service.AgentSvc
			sourceID openapi_types.UUID
			userSvc  *backend.PlannerSvc
		)

		ginkgo.BeforeEach(func() {
			agentSvc = service.DefaultAgentSvc(cfg.AgentAPIUrl)
			userSvc = plannerSvc.WithAuthUser("admin", "admin", "admin@example.com")

			source, err := userSvc.CreateSource("test-source-" + uuid.NewString()[:8])
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to create source")
			sourceID = source.Id
			ginkgo.GinkgoWriter.Printf("Created source: %s\n", sourceID)
		})

		ginkgo.AfterEach(func() {
			if ginkgo.CurrentSpecReport().Failed() {
				ginkgo.GinkgoWriter.Println("Keeping containers running (test failed)")
				return
			}
			ginkgo.GinkgoWriter.Println("Stopping agent...")
			_ = infraManager.RemoveAgent()

			ginkgo.GinkgoWriter.Println("Deleting source...")
			_ = userSvc.RemoveSource(sourceID)
		})

		// Given an agent started in disconnected mode with a valid source ID
		// When the agent mode is switched to connected
		// Then the source should have the agent attached
		ginkgo.It("should register agent with backend after switching from disconnected to connected", func() {
			agentID := uuid.NewString()
			_, err := infraManager.StartAgent(infra.AgentConfig{
				AgentID:        agentID,
				SourceID:       sourceID.String(),
				Mode:           "disconnected",
				ConsoleURL:     cfg.BackendAgentEndpoint,
				APIVersion:     "v2",
				UpdateInterval: "1s",
			})
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to start agent")
			ginkgo.GinkgoWriter.Printf("Agent started with ID: %s\n", agentID)

			gm.Eventually(func() error {
				_, err := agentSvc.Status()
				return err
			}, 30*time.Second, 1*time.Second).Should(gm.BeNil())

			_, err = agentSvc.SetAgentMode("connected")
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to switch mode")

			time.Sleep(5 * time.Second) // allow agent to register with backend
			source, err := userSvc.GetSource(sourceID)
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to get source")

			ginkgo.GinkgoWriter.Printf("Source agent: %+v\n", source.Agent)
			gm.Expect(source.Agent).ToNot(gm.BeNil(), "expected agent to be attached to source after mode switch")
		})

		// Given an agent started in disconnected mode without inventory
		// When the agent mode is switched to connected
		// Then the agent should communicate with backend without errors
		ginkgo.It("should communicate with backend without errors when no inventory is collected", func() {
			agentID := uuid.NewString()
			_, err := infraManager.StartAgent(infra.AgentConfig{
				AgentID:        agentID,
				SourceID:       sourceID.String(),
				Mode:           "disconnected",
				ConsoleURL:     "http://localhost:8081", // through proxy to observe requests
				APIVersion:     "v2",
				UpdateInterval: "1s",
			})
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to start agent")
			ginkgo.GinkgoWriter.Printf("Agent started with ID: %s\n", agentID)

			gm.Eventually(func() error {
				_, err := agentSvc.Status()
				return err
			}, 30*time.Second, 1*time.Second).Should(gm.BeNil())

			// Act - switch to connected mode without collecting inventory
			_, err = agentSvc.SetAgentMode("connected")
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to switch mode")

			time.Sleep(5 * time.Second)

			status, err := agentSvc.Status()
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to get agent status")
			ginkgo.GinkgoWriter.Printf("Agent status: mode=%s, console_connection=%s, error=%s\n",
				status.Mode, status.ConsoleConnection, status.Error)

			gm.Expect(status.Mode).To(gm.Equal("connected"), "expected mode to be connected")
			gm.Expect(status.Error).To(gm.BeEmpty(), "expected no error in agent status")

			reqs := obs.Requests()
			ginkgo.GinkgoWriter.Printf("Observed %d requests to backend\n", len(reqs))
			gm.Expect(reqs).ToNot(gm.BeEmpty(), "expected requests to be made to backend")
		})
	})

	ginkgo.Context("collector", func() {
		var (
			agentSvc *service.AgentSvc
			sourceID openapi_types.UUID
			userSvc  *backend.PlannerSvc
		)

		ginkgo.BeforeEach(func() {
			ginkgo.GinkgoWriter.Println("Starting vcsim...")
			err := infraManager.StartVcsim()
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to start vcsim")
			time.Sleep(1 * time.Second)

			client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			}
			gm.Eventually(func() error {
				resp, err := client.Get(infra.VcsimURL)
				if err != nil {
					return err
				}
				defer func() { _ = resp.Body.Close() }()
				if resp.StatusCode >= 500 {
					return fmt.Errorf("server error: %d", resp.StatusCode)
				}
				return nil
			}, 30*time.Second, 1*time.Second).Should(gm.BeNil())

			agentSvc = service.DefaultAgentSvc(cfg.AgentAPIUrl)
			userSvc = plannerSvc.WithAuthUser("admin", "admin", "admin@example.com")

			source, err := userSvc.CreateSource("test-source-" + uuid.NewString()[:8])
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to create source")
			sourceID = source.Id
			ginkgo.GinkgoWriter.Printf("Created source: %s\n", sourceID)
		})

		ginkgo.AfterEach(func() {
			if ginkgo.CurrentSpecReport().Failed() {
				ginkgo.GinkgoWriter.Println("Keeping containers running (test failed)")
				return
			}
			ginkgo.GinkgoWriter.Println("Stopping agent...")
			_ = infraManager.RemoveAgent()

			ginkgo.GinkgoWriter.Println("Deleting source...")
			_ = userSvc.RemoveSource(sourceID)

			ginkgo.GinkgoWriter.Println("Stopping vcsim...")
			_ = infraManager.StopVcsim()
		})

		startConnectedAgentAndCollect := func(agentID string) {
			_, err := infraManager.StartAgent(infra.AgentConfig{
				AgentID:        agentID,
				SourceID:       sourceID.String(),
				Mode:           "connected",
				ConsoleURL:     "http://localhost:8081",
				APIVersion:     "v2",
				UpdateInterval: "1s",
			})
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to start agent")
			ginkgo.GinkgoWriter.Printf("Agent started with ID: %s\n", agentID)

			gm.Eventually(func() error {
				_, err := agentSvc.Status()
				return err
			}, 30*time.Second, 1*time.Second).Should(gm.BeNil())

			ginkgo.GinkgoWriter.Println("Storing credentials...")
			_, err = agentSvc.StoreCredentials(infra.VcsimURL, infra.VcsimUsername, infra.VcsimPassword)
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to store credentials")

			ginkgo.GinkgoWriter.Println("Starting collector...")
			_, err = agentSvc.StartCollector()
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to start collector")

			gm.Eventually(func() int {
				collections, err := agentSvc.ListCollections()
				if err != nil {
					return 0
				}
				return len(collections.Collections)
			}, 120*time.Second, 2*time.Second).Should(gm.BeNumerically(">", 0), "expected at least 1 collection")
		}

		// Given an agent in connected mode with a successful collection
		// When a group is created that matches collected VMs
		// Then the group's inventory subset should be pushed to the backend
		ginkgo.It("should push inventory to backend after successful collection", func() {
			agentID := uuid.NewString()
			startConnectedAgentAndCollect(agentID)

			pageSize := 1
			vms, err := agentSvc.ListLatestVMs(&service.VMListParams{PageSize: &pageSize})
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to list VMs")
			gm.Expect(vms.VirtualMachines).ToNot(gm.BeEmpty(), "expected at least one collected VM")
			cluster := vms.VirtualMachines[0].Cluster

			ginkgo.GinkgoWriter.Println("Creating group matching collected VMs...")
			group, err := agentSvc.CreateGroup("connected-e2e-group", fmt.Sprintf("cluster = '%s'", cluster), "connected env test group")
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to create group")
			defer func() { _, _ = agentSvc.DeleteGroup(group.Id) }()

			// Assert - group inventory subset should be pushed to the backend via console
			gm.Eventually(func() bool {
				for _, r := range obs.Requests() {
					if strings.Contains(r.Request.URL.Path, "subset") && strings.Contains(r.Request.URL.Path, group.Id) {
						return true
					}
				}
				return false
			}, 15*time.Second, 1*time.Second).Should(gm.BeTrue(), "expected group inventory subset to be pushed to backend")
		})

		// Given an agent that switches to disconnected mode before collecting
		// When the inventory is collected and manually uploaded to the backend
		// Then the source should have the inventory populated
		ginkgo.It("should manually upload inventory from disconnected agent to backend", func() {
			agentID := uuid.NewString()
			_, err := infraManager.StartAgent(infra.AgentConfig{
				AgentID:        agentID,
				SourceID:       sourceID.String(),
				Mode:           "connected",
				ConsoleURL:     "http://localhost:8081",
				APIVersion:     "v2",
				UpdateInterval: "1s",
			})
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to start agent")
			ginkgo.GinkgoWriter.Printf("Agent started with ID: %s\n", agentID)

			gm.Eventually(func() error {
				_, err := agentSvc.Status()
				return err
			}, 30*time.Second, 1*time.Second).Should(gm.BeNil())

			ginkgo.GinkgoWriter.Println("Switching agent to disconnected mode...")
			status, err := agentSvc.SetAgentMode("disconnected")
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to switch to disconnected mode")
			gm.Expect(status.Mode).To(gm.Equal("disconnected"))

			ginkgo.GinkgoWriter.Println("Storing credentials and starting collector...")
			_, err = agentSvc.StoreCredentials(infra.VcsimURL, infra.VcsimUsername, infra.VcsimPassword)
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to store credentials")
			_, err = agentSvc.StartCollector()
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to start collector")

			gm.Eventually(func() int {
				collections, err := agentSvc.ListCollections()
				if err != nil {
					return 0
				}
				return len(collections.Collections)
			}, 120*time.Second, 2*time.Second).Should(gm.BeNumerically(">", 0), "expected at least 1 collection")

			inv, err := agentSvc.Inventory()
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to get inventory")
			gm.Expect(inv).ToNot(gm.BeNil(), "expected inventory to be available")
			ginkgo.GinkgoWriter.Printf("Collected inventory with vcenter_id: %s\n", inv.VcenterId)

			ginkgo.GinkgoWriter.Println("Manually uploading inventory to backend...")
			err = userSvc.UpdateSource(sourceID, &v1alpha1.UpdateInventory{
				AgentId:   uuid.MustParse(agentID),
				Inventory: *inv,
			})
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to upload inventory to backend")

			source, err := userSvc.GetSource(sourceID)
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to get source")
			gm.Expect(source.Inventory).ToNot(gm.BeNil(), "expected inventory to be populated")
			ginkgo.GinkgoWriter.Printf("Source inventory after upload: vcenter_id=%s\n", source.Inventory.VcenterId)
			gm.Expect(source.Inventory.VcenterId).To(gm.Equal(inv.VcenterId), "expected vcenter_id to match")
		})

		// Given an agent in connected mode that pushed a group inventory subset
		// When the agent is restarted
		// Then the subset should not be redelivered and the agent should resume reporting status
		ginkgo.It("should preserve inventory on backend after agent restart", func() {
			agentID := uuid.NewString()
			startConnectedAgentAndCollect(agentID)

			pageSize := 1
			vms, err := agentSvc.ListLatestVMs(&service.VMListParams{PageSize: &pageSize})
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to list VMs")
			gm.Expect(vms.VirtualMachines).ToNot(gm.BeEmpty(), "expected at least one collected VM")
			cluster := vms.VirtualMachines[0].Cluster

			group, err := agentSvc.CreateGroup("restart-e2e-group", fmt.Sprintf("cluster = '%s'", cluster), "restart test group")
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to create group")
			defer func() { _, _ = agentSvc.DeleteGroup(group.Id) }()

			subsetRequestCount := func() int {
				count := 0
				for _, r := range obs.Requests() {
					if strings.Contains(r.Request.URL.Path, "subset") && strings.Contains(r.Request.URL.Path, group.Id) {
						count++
					}
				}
				return count
			}

			gm.Eventually(subsetRequestCount, 15*time.Second, 1*time.Second).Should(gm.BeNumerically(">", 0),
				"expected group inventory subset to be pushed before restart")
			countBeforeRestart := subsetRequestCount()
			totalBeforeRestart := len(obs.Requests())

			ginkgo.GinkgoWriter.Println("Restarting agent...")
			err = infraManager.RestartAgent()
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to restart agent")

			gm.Eventually(func() error {
				_, err := agentSvc.Status()
				return err
			}, 30*time.Second, 1*time.Second).Should(gm.BeNil())

			// Assert - agent resumes reporting status after restart
			gm.Eventually(func() int {
				return len(obs.Requests())
			}, 15*time.Second, 1*time.Second).Should(gm.BeNumerically(">", totalBeforeRestart),
				"expected agent to resume sending requests after restart")

			// Assert - the already-delivered subset event is not redelivered
			time.Sleep(3 * time.Second)
			gm.Expect(subsetRequestCount()).To(gm.Equal(countBeforeRestart),
				"expected no duplicate subset delivery after restart")
		})
	})

	ginkgo.Context("console connectivity", func() {
		var (
			agentSvc *service.AgentSvc
			sourceID openapi_types.UUID
			userSvc  *backend.PlannerSvc
		)

		ginkgo.BeforeEach(func() {
			agentSvc = service.DefaultAgentSvc(cfg.AgentAPIUrl)
			userSvc = plannerSvc.WithAuthUser("admin", "admin", "admin@example.com")

			source, err := userSvc.CreateSource("test-source-" + uuid.NewString()[:8])
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to create source")
			sourceID = source.Id
			ginkgo.GinkgoWriter.Printf("Created source: %s\n", sourceID)
		})

		ginkgo.AfterEach(func() {
			// The backend is shared by every spec in "Connected env v2", so it must be
			// restored regardless of pass/fail before the next ordered spec runs.
			ginkgo.GinkgoWriter.Println("Ensuring backend is running...")
			_ = infraManager.StartBackend()
			gm.Eventually(func() error {
				resp, err := http.DefaultClient.Get(cfg.BackendAgentEndpoint + "/health")
				if err != nil {
					return err
				}
				defer func() { _ = resp.Body.Close() }()
				if resp.StatusCode >= 500 {
					return fmt.Errorf("server error: %d", resp.StatusCode)
				}
				return nil
			}, 30*time.Second, 1*time.Second).Should(gm.BeNil())

			if ginkgo.CurrentSpecReport().Failed() {
				ginkgo.GinkgoWriter.Println("Keeping containers running (test failed)")
				return
			}
			ginkgo.GinkgoWriter.Println("Stopping agent...")
			_ = infraManager.RemoveAgent()

			ginkgo.GinkgoWriter.Println("Deleting source...")
			_ = userSvc.RemoveSource(sourceID)
		})

		// Given an agent connected to the console through a proxy
		// When the console backend becomes unreachable and then recovers
		// Then the agent should report disconnected with an error, then reconnect with no error
		ginkgo.It("should report disconnected on transient console errors and reconnect once the console recovers", func() {
			agentID := uuid.NewString()
			_, err := infraManager.StartAgent(infra.AgentConfig{
				AgentID:        agentID,
				SourceID:       sourceID.String(),
				Mode:           "connected",
				ConsoleURL:     "http://localhost:8081",
				APIVersion:     "v2",
				UpdateInterval: "1s",
			})
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to start agent")
			ginkgo.GinkgoWriter.Printf("Agent started with ID: %s\n", agentID)

			gm.Eventually(func() error {
				_, err := agentSvc.Status()
				return err
			}, 30*time.Second, 1*time.Second).Should(gm.BeNil())

			gm.Eventually(func() string {
				status, err := agentSvc.Status()
				if err != nil {
					return "error"
				}
				return status.ConsoleConnection
			}, 30*time.Second, 1*time.Second).Should(gm.Equal("connected"), "expected agent to connect to console before the fault is injected")

			// Act - take the console backend down to force transient errors
			ginkgo.GinkgoWriter.Println("Stopping backend to simulate transient console errors...")
			err = infraManager.StopBackend()
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to stop backend")

			// Assert - agent should report disconnected
			gm.Eventually(func() string {
				status, err := agentSvc.Status()
				if err != nil {
					return "error"
				}
				ginkgo.GinkgoWriter.Printf("Agent status while backend down: mode=%s, console_connection=%s, error=%s\n",
					status.Mode, status.ConsoleConnection, status.Error)
				return status.ConsoleConnection
			}, 30*time.Second, 1*time.Second).Should(gm.Equal("disconnected"))

			// Assert - and the error field should eventually be populated too
			gm.Eventually(func() string {
				status, err := agentSvc.Status()
				if err != nil {
					return ""
				}
				return status.Error
			}, 10*time.Second, 1*time.Second).ShouldNot(gm.BeEmpty(), "expected agent status to report an error while console is unreachable")

			// Act - bring the console backend back up
			ginkgo.GinkgoWriter.Println("Restarting backend to simulate console recovery...")
			err = infraManager.StartBackend()
			gm.Expect(err).ToNot(gm.HaveOccurred(), "failed to restart backend")

			gm.Eventually(func() error {
				resp, err := http.DefaultClient.Get(cfg.BackendAgentEndpoint + "/health")
				if err != nil {
					return err
				}
				defer func() { _ = resp.Body.Close() }()
				if resp.StatusCode >= 500 {
					return fmt.Errorf("server error: %d", resp.StatusCode)
				}
				return nil
			}, 30*time.Second, 1*time.Second).Should(gm.BeNil())

			// Assert - agent should recover to connected with no error
			gm.Eventually(func() string {
				status, err := agentSvc.Status()
				if err != nil {
					return "error"
				}
				return status.ConsoleConnection
			}, 30*time.Second, 1*time.Second).Should(gm.Equal("connected"))

			gm.Eventually(func() string {
				status, err := agentSvc.Status()
				if err != nil {
					return "error"
				}
				return status.Error
			}, 10*time.Second, 1*time.Second).Should(gm.BeEmpty(), "expected no error after console recovers")
		})
	})
})
