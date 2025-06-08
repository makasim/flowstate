package recovery

import (
	"strconv"
	"time"

	"github.com/makasim/flowstate"
)

var EnabledAnnotation = `flowstate.recovery.enabled`

func Disable(stateCtx *flowstate.StateCtx) {
	stateCtx.Current.SetAnnotation(EnabledAnnotation, "false")
}

func enabled(state flowstate.State) bool {
	return state.Annotations[EnabledAnnotation] != "false"
}

var AttemptAnnotation = `flowstate.recovery.attempt`

func Attempt(state flowstate.State) int {
	attempt, _ := strconv.Atoi(state.Transition.Annotations[AttemptAnnotation])
	return attempt
}

func setAttempt(stateCtx *flowstate.StateCtx, attempt int) {
	stateCtx.Current.Transition.SetAnnotation(AttemptAnnotation, strconv.Itoa(attempt))
}

var DefaultMaxAttempts = 3
var MaxAttemptsAnnotation = `flowstate.recovery.max_attempts`

func MaxAttempts(state flowstate.State) int {
	attempt, _ := strconv.Atoi(state.Annotations[MaxAttemptsAnnotation])
	if attempt <= 0 {
		return DefaultMaxAttempts
	}

	return attempt
}

func SetMaxAttempts(stateCtx *flowstate.StateCtx, attempts int) {
	if attempts <= 0 {
		attempts = DefaultMaxAttempts
	}

	stateCtx.Current.SetAnnotation(MaxAttemptsAnnotation, strconv.Itoa(attempts))
}

var DefaultRetryAfter = time.Minute * 2
var MinRetryAfter = time.Minute
var MaxRetryAfter = time.Minute * 5
var RetryAfterAnnotation = `flowstate.recovery.retry_after`

func SetRetryAfter(stateCtx *flowstate.StateCtx, retryAfter time.Duration) {
	if retryAfter < MinRetryAfter {
		retryAfter = MinRetryAfter
	}
	if retryAfter > MaxRetryAfter {
		retryAfter = MaxRetryAfter
	}

	stateCtx.Current.SetAnnotation(RetryAfterAnnotation, retryAfter.String())
}

func retryAt(state flowstate.State) time.Time {
	retryAfterStr := state.Annotations[RetryAfterAnnotation]
	if retryAfterStr == "" {
		retryAfterStr = DefaultRetryAfter.String()
	}

	retryAfter, err := time.ParseDuration(retryAfterStr)
	if err != nil {
		return state.CommittedAt().Add(MinRetryAfter)
	}

	if retryAfter < MinRetryAfter {
		retryAfter = MinRetryAfter
	}
	if retryAfter > MaxRetryAfter {
		retryAfter = MaxRetryAfter
	}

	return state.CommittedAt().Add(retryAfter)
}
