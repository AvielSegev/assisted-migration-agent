package scheduler

import (
	"context"
	"fmt"
	"sync"
)

type queue[T any] []T

func (wq *queue[T]) Len() int { return len(*wq) }

func (wq *queue[T]) Pop() T {
	old := *wq
	x := old[0]
	*wq = old[1:]
	return x
}

func (wq *queue[T]) Push(t T) {
	*wq = append(*wq, t)
}

type workRequest struct {
	fn  Work[any]
	c   chan Result[any]
	ctx context.Context
}

type worker struct {
	done chan any
	wg   *sync.WaitGroup
}

func (w worker) Work(r workRequest) {
	defer func() {
		if rec := recover(); rec != nil {
			r.c <- Result[any]{Err: fmt.Errorf("worker panicked: %v", rec)}
		}
		w.done <- struct{}{}
		w.wg.Done()
	}()

	v, err := r.fn(r.ctx)
	r.c <- Result[any]{Data: v, Err: err}
}

func newWorker(done chan any, wg *sync.WaitGroup) worker {
	return worker{done: done, wg: wg}
}

type Scheduler struct {
	workers    *queue[worker]
	workQueue  *queue[workRequest]
	close      chan any
	done       chan any
	work       chan workRequest
	mainCtx    context.Context
	mainCancel context.CancelFunc
	wg         sync.WaitGroup
	once       sync.Once
}

func NewScheduler(nbWorkers int) *Scheduler {
	done := make(chan any, nbWorkers)
	ctx, cancel := context.WithCancel(context.Background())
	s := &Scheduler{
		workers:    &queue[worker]{},
		workQueue:  &queue[workRequest]{},
		close:      make(chan any),
		done:       done,
		work:       make(chan workRequest),
		mainCtx:    ctx,
		mainCancel: cancel,
	}
	for range nbWorkers {
		s.workers.Push(newWorker(done, &s.wg))
	}
	go s.run()
	return s
}

func (s *Scheduler) AddWork(w Work[any]) *Future[Result[any]] {
	c := make(chan Result[any], 1)
	ctx, cancel := context.WithCancel(s.mainCtx)

	select {
	case <-s.mainCtx.Done():
		// we're closing here so send a result with an error
		c <- Result[any]{Err: context.Canceled}
	case s.work <- workRequest{w, c, ctx}:
	}

	return NewFuture(c, cancel)
}

func (s *Scheduler) Close() {
	s.once.Do(func() {
		s.mainCancel()
		s.close <- struct{}{}
		<-s.done
	})
}

func (s *Scheduler) run() {
	defer close(s.done)
	for {
		select {
		case w := <-s.work:
			s.workQueue.Push(w)
			s.dispatch()
		case <-s.done:
			s.workers.Push(newWorker(s.done, &s.wg))
			s.dispatch()
		case <-s.close:
			s.wg.Wait()
			return
		}
	}
}

// dispatch drains the workQueue as much as possible
// based on available workers
func (s *Scheduler) dispatch() {
	for s.workers.Len() > 0 && s.workQueue.Len() > 0 {
		r := s.workQueue.Pop()
		worker := s.workers.Pop()
		s.wg.Add(1)
		go worker.Work(r)
	}
}
