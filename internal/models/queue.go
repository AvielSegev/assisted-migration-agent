package models

import "context"

type Work[T any] func(ctx context.Context) (T, error)

type Queue[T any] []T

func (wq *Queue[T]) Len() int { return len(*wq) }

func (wq *Queue[T]) Pop() T {
	old := *wq
	x := old[0]
	*wq = old[1:]
	return x
}

func (wq *Queue[T]) Push(t T) {
	*wq = append(*wq, t)
}

type Result[T any] struct {
	Data T
	Err  error
}
