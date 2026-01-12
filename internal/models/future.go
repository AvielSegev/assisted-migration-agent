package models

import (
	"context"
)

type Future[T any] struct {
	input  chan T
	cancel context.CancelFunc
}

func NewFuture[T any](input chan T, cancel context.CancelFunc) *Future[T] {
	f := &Future[T]{
		input:  input,
		cancel: cancel,
	}

	return f
}

func (f *Future[T]) C() chan T {
	return f.input
}

func (f *Future[T]) Stop() {
	f.cancel()
}
