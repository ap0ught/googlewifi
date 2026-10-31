package monitor

import (
	"testing"
	"time"
)

func TestEvaluatePointDoesNotTreatMissingWANAsUnhealthy(t *testing.T) {
	got := Evaluate(StatusInput{
		HTTPStatus: 200,
		Role:       "leaf",
		LED:        "CONNECTED",
		PingLoss:   0,
		PingAvgMS:  6,
	})
	if got.State != StateOK {
		t.Fatalf("state = %s, want OK; reason=%s", got.State, got.Reason)
	}
}

func TestEvaluateDetectsUnreachablePoint(t *testing.T) {
	got := Evaluate(StatusInput{Err: errUnreachable, Role: "leaf"})
	if got.State != StateCritical {
		t.Fatalf("state = %s, want CRITICAL", got.State)
	}
}

func TestCondenseOnlyRecordsStateChanges(t *testing.T) {
	now := time.Date(2026, 10, 31, 1, 0, 0, 0, time.UTC)
	state := &HostState{}
	ok := Result{State: StateOK, Reason: "connected"}

	if event := Condense(state, ok, now); event == nil {
		t.Fatal("first result should create an event")
	}
	if event := Condense(state, ok, now.Add(time.Minute)); event != nil {
		t.Fatal("repeated OK result should be condensed")
	}
	failed := Result{State: StateCritical, Reason: "status request failed"}
	if event := Condense(state, failed, now.Add(2*time.Minute)); event == nil {
		t.Fatal("state change should create an event")
	} else if event.Previous != StateOK || event.State != StateCritical {
		t.Fatalf("event = %+v, want OK -> CRITICAL", event)
	}
	if state.Consecutive != 1 {
		t.Fatalf("consecutive = %d, want 1", state.Consecutive)
	}
}

var errUnreachable = &testError{"unreachable"}

type testError struct{ text string }

func (e *testError) Error() string { return e.text }
