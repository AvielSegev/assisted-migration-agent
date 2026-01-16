package scheduler_test

import (
	"context"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubev2v/assisted-migration-agent/internal/models"
	"github.com/kubev2v/assisted-migration-agent/pkg/scheduler"
)

var _ = Describe("Scheduler", func() {
	var s *scheduler.Scheduler

	AfterEach(func() {
		if s != nil {
			s.Close()
		}
	})

	Describe("AddWork", func() {
		It("should add work and return a future", func() {
			s = scheduler.NewScheduler(1)

			work := func(ctx context.Context) (any, error) {
				return "done", nil
			}

			future := s.AddWork(work)
			Expect(future).NotTo(BeNil())

			var result models.Result[any]
			Eventually(future.C(), 2*time.Second).Should(Receive(&result))
			Expect(result.Data).To(Equal("done"))
		})
	})

	Describe("Run work", func() {
		It("should execute multiple work items", func() {
			s = scheduler.NewScheduler(2)

			results := make(chan int, 3)
			for i := range 3 {
				idx := i
				work := func(ctx context.Context) (any, error) {
					results <- idx
					return idx, nil
				}
				s.AddWork(work)
			}

			Eventually(func() int {
				return len(results)
			}, 2*time.Second, 100*time.Millisecond).Should(Equal(3))
		})
	})

	Describe("Cancel work", func() {
		It("should cancel work via future.Stop()", func() {
			s = scheduler.NewScheduler(1)

			cancelled := make(chan bool, 1)
			work := func(ctx context.Context) (any, error) {
				select {
				case <-ctx.Done():
					cancelled <- true
					return nil, ctx.Err()
				case <-time.After(5 * time.Second):
					return "completed", nil
				}
			}

			future := s.AddWork(work)
			time.Sleep(100 * time.Millisecond)
			future.Stop()

			Eventually(cancelled, 2*time.Second).Should(Receive(BeTrue()))
		})

		It("should cancel work when scheduler is closed", func() {
			s = scheduler.NewScheduler(1)

			cancelled := make(chan bool, 1)
			work := func(ctx context.Context) (any, error) {
				select {
				case <-ctx.Done():
					cancelled <- true
					return nil, ctx.Err()
				case <-time.After(5 * time.Second):
					return "completed", nil
				}
			}

			s.AddWork(work)
			time.Sleep(100 * time.Millisecond)
			s.Close()
			s = nil // prevent AfterEach from closing again

			Eventually(cancelled, 2*time.Second).Should(Receive(BeTrue()))
		})
	})

	Describe("Goroutine cleanup", func() {
		It("should not leak goroutines after Close under load", func() {
			base := runtime.NumGoroutine()
			s = scheduler.NewScheduler(4)

			work := func(ctx context.Context) (any, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			}

			for i := 0; i < 200; i++ {
				s.AddWork(work)
			}

			time.Sleep(100 * time.Millisecond)
			s.Close()
			s = nil // prevent AfterEach from closing again

			Eventually(func() int {
				return runtime.NumGoroutine()
			}, 5*time.Second, 100*time.Millisecond).Should(BeNumerically("<=", base+10))
		})
	})

	Describe("Close behavior", func() {
		It("should return canceled when AddWork is called after Close", func() {
			s = scheduler.NewScheduler(1)
			s.Close()

			future := s.AddWork(func(ctx context.Context) (any, error) {
				return "done", nil
			})

			var result models.Result[any]
			Eventually(future.C(), 1*time.Second).Should(Receive(&result))
			Expect(result.Err).To(MatchError(context.Canceled))
		})

		It("should wait for in-flight work to finish on Close", func() {
			s = scheduler.NewScheduler(1)

			started := make(chan struct{})
			unblock := make(chan struct{})
			work := func(ctx context.Context) (any, error) {
				close(started)
				<-unblock
				return "done", nil
			}

			s.AddWork(work)
			Eventually(started, 1*time.Second).Should(BeClosed())

			closeDone := make(chan struct{})
			go func() {
				s.Close()
				close(closeDone)
			}()

			Consistently(closeDone, 200*time.Millisecond).ShouldNot(BeClosed())
			close(unblock)
			Eventually(closeDone, 1*time.Second).Should(BeClosed())
			s = nil // prevent AfterEach from closing again
		})
	})
})
