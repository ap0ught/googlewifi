package monitor

import (
	"fmt"
	"time"
)

type State string

const (
	StateOK       State = "OK"
	StateWarning  State = "WARNING"
	StateCritical State = "CRITICAL"
	StateUnknown  State = "UNKNOWN"
)

type StatusInput struct {
	Err        error
	HTTPStatus int
	Role       string
	LED        string
	PingLoss   float64
	PingAvgMS  float64
}

type Result struct {
	State  State  `json:"state"`
	Reason string `json:"reason"`
}

type HostState struct {
	State       State     `json:"state,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	Consecutive int       `json:"consecutive"`
	Checks      int       `json:"checks"`
	LastCheck   time.Time `json:"last_check"`
	LastChange  time.Time `json:"last_change"`
}

type Event struct {
	At       time.Time `json:"at"`
	Previous State     `json:"previous,omitempty"`
	State    State     `json:"state"`
	Reason   string    `json:"reason"`
}

func Evaluate(in StatusInput) Result {
	if in.Err != nil || in.HTTPStatus < 200 || in.HTTPStatus >= 300 {
		return Result{State: StateCritical, Reason: "status request failed"}
	}
	if in.Role == "leaf" && in.LED != "CONNECTED" {
		return Result{State: StateCritical, Reason: fmt.Sprintf("mesh point LED is %s", in.LED)}
	}
	if in.Role == "root" && in.LED != "CONNECTED" {
		return Result{State: StateWarning, Reason: fmt.Sprintf("root LED is %s", in.LED)}
	}
	if in.Role == "root" && in.PingLoss >= 50 {
		return Result{State: StateCritical, Reason: fmt.Sprintf("root ping loss %.1f%%", in.PingLoss)}
	}
	if in.Role == "leaf" && in.PingLoss >= 50 {
		return Result{State: StateCritical, Reason: fmt.Sprintf("mesh point ping loss %.1f%%", in.PingLoss)}
	}
	if in.PingLoss > 0 {
		return Result{State: StateWarning, Reason: fmt.Sprintf("ping loss %.1f%%", in.PingLoss)}
	}
	if in.Role == "leaf" && in.PingAvgMS >= 100 {
		return Result{State: StateWarning, Reason: fmt.Sprintf("mesh point latency %.1f ms", in.PingAvgMS)}
	}
	return Result{State: StateOK, Reason: "connected"}
}

func Condense(state *HostState, result Result, now time.Time) *Event {
	previous := state.State
	state.Checks++
	state.LastCheck = now
	state.Reason = result.Reason
	if previous == result.State {
		state.Consecutive++
		return nil
	}
	state.State = result.State
	state.Consecutive = 1
	state.LastChange = now
	return &Event{At: now, Previous: previous, State: result.State, Reason: result.Reason}
}
