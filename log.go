package flowstate

func logWithStateCtx(stateCtx *StateCtx, args []any) []any {
	args = append(args, []any{
		"id", stateCtx.Current.ID,
		"rev", stateCtx.Current.Rev,
	}...)

	currTs := stateCtx.Current.Transition
	
	if currTs.Annotations[StateAnnotation] != `` {
		args = append(args, "state", currTs.Annotations[StateAnnotation])
	}

	if currTs.Annotations[DelayDurationAnnotation] != `` {
		args = append(args, "delayed", "true")
	}

	if currTs.Annotations[RecoveryAttemptAnnotation] != `` {
		args = append(args, "recovered", currTs.Annotations[RecoveryAttemptAnnotation])
	}

	return args
}
