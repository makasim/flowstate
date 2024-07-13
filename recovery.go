package flowstate

import "strconv"

var RecoveryAttemptAnnotation = `flowstate.recovery_attempt`

func RecoveryAttempt(state State) int {
	attempt, _ := strconv.Atoi(state.Transition.Annotations[RecoveryAttemptAnnotation])
	return attempt
}
