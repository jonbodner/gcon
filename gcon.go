package gcon

import (
	"context"
	"errors"
	"sync"
)

// Func represents any function that returns a Promise when passed to Run or Then.
type Func[T, V any] func(context.Context, T) (V, error)

var (
	// ErrIncomplete is returned when GetNow is invoked and the Func associated with the Promise hasn't completed.
	ErrIncomplete = errors.New("incomplete")
)

// Promise represents a potential or actual result from running a Func.
type Promise[V any] struct {
	val  V
	err  error
	done <-chan struct{}
}

// Get returns the value and the error (if any) for the Promise. Get waits until the Func associated with this
// Promise has completed. If the Func has completed, Get returns immediately.
func (p *Promise[V]) Get() (V, error) {
	<-p.done
	return p.val, p.err
}

// GetNow returns the value and the error (if any) for the Promise. If the Func associated with this Promise has
// not completed, GetNow returns the zero value for the return type and ErrIncomplete.
func (p *Promise[V]) GetNow() (V, error) {
	select {
	case <-p.done:
		return p.val, p.err
	default:
		var zero V
		return zero, ErrIncomplete
	}
}

// Run produces a Promise for the supplied Func, evaluating the supplied context.Context and data. The Promise is
// returned immediately, no matter how long it takes for the Func to complete processing.
func Run[T, V any](ctx context.Context, t T, f Func[T, V]) *Promise[V] {
	done := make(chan struct{})
	p := Promise[V]{
		done: done,
	}
	go func() {
		defer close(done)
		p.val, p.err = f(ctx, t)
	}()
	return &p
}

// Waiter defines an interface for the parameters to the Wait function.
type Waiter interface {
	Wait() error
}

// Wait allows a Promise to implement the Waiter interface. It is similar to Get, but only returns the error.
func (p *Promise[V]) Wait() error {
	<-p.done
	return p.err
}

// Wait takes in zero or more Waiter instances and paused until one returns an error or all of them complete successfully.
// It returns the first error from a Waiter or nil, if no Waiter returns an error.
func Wait(ws ...Waiter) error {
	var wg sync.WaitGroup
	wg.Add(len(ws))
	errChan := make(chan error, len(ws))
	done := make(chan struct{})
	for _, w := range ws {
		go func(w Waiter) {
			defer wg.Done()
			err := w.Wait()
			if err != nil {
				errChan <- err
			}
		}(w)
	}
	go func() {
		defer close(done)
		wg.Wait()
	}()
	select {
	case err := <-errChan:
		return err
	case <-done:
	}
	return nil
}

// WithCancellation takes in a Func and returns a Func that implements the passed-in Func's behavior, but adds context cancellation.
func WithCancellation[T, V any](f Func[T, V]) Func[T, V] {
	return func(ctx context.Context, t T) (V, error) {
		done := make(chan struct{})
		var val V
		var err error
		go func() {
			defer close(done)
			val, err = f(ctx, t)
		}()
		select {
		case <-ctx.Done():
			var zero V
			return zero, ctx.Err()
		case <-done:
		}
		return val, err
	}
}

// Then produces a Promise for the supplied Func, evaluating the supplied context.Context and Promise. The returned Promise is
// returned immediately, no matter how long it takes for the Func to complete processing. If the supplied Promise returns a
// non-nil error, the error is propagated to the returned Promise and the passed-in Func is not run.
func Then[T, V any](ctx context.Context, p *Promise[T], f Func[T, V]) *Promise[V] {
	done := make(chan struct{})
	out := Promise[V]{
		done: done,
	}
	go func() {
		defer close(done)
		val, err := p.Get()
		if err != nil {
			out.err = err
			return
		}
		val2, err := f(ctx, val)
		out.val = val2
		out.err = err
	}()
	return &out
}
