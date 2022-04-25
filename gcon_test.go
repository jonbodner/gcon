package gcon

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

func doubler(ctx context.Context, i int) (int, error) {
	return i * 2, nil
}

func tripler(ctx context.Context, i int) (int, error) {
	return i * 3, nil
}

func fromString(ctx context.Context, s string) (int, error) {
	return strconv.Atoi(s)
}

func toString(ctx context.Context, i int) (string, error) {
	return strconv.Itoa(i), nil
}

func timed(ctx context.Context, d time.Duration) (int, error) {
	time.Sleep(d)
	return int(d), nil
}

var ErrFail = errors.New("fail")

func alwaysErr(ctx context.Context, i int) (int, error) {
	return 0, ErrFail
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	// runs instantly and works
	p := Run(ctx, 10, doubler)
	v, err := p.Get()
	if v != 20 {
		t.Error("Expected 20, got ", v)
	}
	if err != nil {
		t.Error("Expected no error, got ", err)
	}
	// runs with a pause and works
	p2 := Run(ctx, time.Second, timed)
	v2, err := p2.Get()
	if v2 != 1000000000 {
		t.Error("Expected 1000000000, got ", v2)
	}
	if err != nil {
		t.Error("Expected no error, got ", err)
	}
	// runs and returns an error
	p3 := Run(ctx, 1, alwaysErr)
	v3, err := p3.Get()
	if v3 != 0 {
		t.Error("Expected 0, got ", v3)
	}
	if !errors.Is(err, ErrFail) {
		t.Error("expected ErrFail, got ", err)
	}
}

func TestPromise_GetNow(t *testing.T) {
	ctx := context.Background()
	// runs with a pause see error, and then wait and see correct
	p := Run(ctx, time.Second, timed)
	v, err := p.GetNow()
	if v != 0 {
		t.Error("Expected 0, got ", v)
	}
	if !errors.Is(err, ErrIncomplete) {
		t.Error("Expected ErrIncomplete, got ", err)
	}
	v, err = p.Get()
	if v != 1000000000 {
		t.Error("Expected 1000000000, got ", v)
	}
	if err != nil {
		t.Error("Expected no error, got ", err)
	}
	v, err = p.GetNow()
	if v != 1000000000 {
		t.Error("Expected 1000000000, got ", v)
	}
	if err != nil {
		t.Error("Expected no error, got ", err)
	}
}

func TestWait(t *testing.T) {
	ctx := context.Background()
	//everything finishes
	p := Run(ctx, "100", fromString)
	p2 := Run(ctx, 200, doubler)
	err := Wait(p, p2)
	if err != nil {
		t.Error("expected no error, got ", err)
	}
	v, err := p.Get()
	if v != 100 {
		t.Error("expected 100, got ", v)
	}
	if err != nil {
		t.Error("expected no error, got ", err)
	}
	v2, err := p2.Get()
	if v2 != 400 {
		t.Error("expected 400, got ", v2)
	}
	if err != nil {
		t.Error("expected no error, got ", err)
	}

	// one errors out
	p3 := Run(ctx, 2*time.Second, timed)
	p4 := Run(ctx, 0, alwaysErr)
	err = Wait(p3, p4)
	if !errors.Is(err, ErrFail) {
		t.Error("expected ErrFail, got ", err)
	}
	// won't be done, returns ErrIncomplete
	v3, err := p3.GetNow()
	if v3 != 0 {
		t.Error("expected 0, got ", v3)
	}
	if !errors.Is(err, ErrIncomplete) {
		t.Error("expected ErrIncomplete, got ", err)
	}
	v4, err := p4.Get()
	if v4 != 0 {
		t.Error("expected 0, got ", v4)
	}
	if !errors.Is(err, ErrFail) {
		t.Error("expected ErrFail, got ", err)
	}
}

func TestWithCancellation(t *testing.T) {
	timedWC := WithCancellation(timed)
	//works without cancellation
	ctx := context.Background()
	p := Run(ctx, time.Second, timedWC)
	v, err := p.Get()
	if v != 1000000000 {
		t.Error("Expected 1000000000, got ", v)
	}
	if err != nil {
		t.Error("Expected no error, got ", err)
	}
	// now gets cancelled
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	p2 := Run(ctx, time.Second, timedWC)
	v2, err := p2.Get()
	if v2 != 0 {
		t.Error("Expected 0, got ", v2)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("Expected DeadlineExceeded, got ", err)
	}
}

func TestThen(t *testing.T) {
	ctx := context.Background()
	// no error
	p := Run(ctx, 10, doubler)
	p2 := Then(ctx, p, toString)

	v2, err := p2.Get()
	if v2 != "20" {
		t.Error("expected \"20\", got", v2)
	}
	if err != nil {
		t.Error("Expected no error, got ", err)
	}

	// with error
	p3 := Run(ctx, 0, alwaysErr)
	p4 := Then(ctx, p3, toString)
	v4, err := p4.Get()
	if v4 != "" {
		t.Error("expected \"\", got ", v4)
	}
	if !errors.Is(err, ErrFail) {
		t.Error("expected ErrFail, got", err)
	}
}
