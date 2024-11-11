package flowstate

import (
	"fmt"
	"log/slog"
)

func logExecute(stateCtx *StateCtx, l *slog.Logger) {
	args := []any{
		"sess", stateCtx.sessID,
		"flow", stateCtx.Current.Transition.ToID,
		"id", stateCtx.Current.ID,
		"rev", stateCtx.Current.Rev,
	}

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

	l.Info("execute flow", args...)
}

func logDo(cmd0 Command, l *slog.Logger) {
	args := []any{"sess", cmd0.SessID()}

	switch cmd := cmd0.(type) {
	case *CommitCommand:
		//for _, c := range cmd.Commands {
		//	log.Printf("commit: %T", c)
		//}
		args = append(args, "cmd", "commit", "len", len(cmd.Commands))
	case *CommitStateCtxCommand:
		args = append(args, "cmd", "commit_state_ctx", "id", cmd.StateCtx.Current.ID, "rev", cmd.StateCtx.Current.Rev)
	case *TransitCommand:
		args = append(args, "cmd", "transit", "id", cmd.StateCtx.Current.ID, "rev", cmd.StateCtx.Current.Rev, "to", cmd.FlowID)
	case *PauseCommand:
		args = append(args, "cmd", "pause", "id", cmd.StateCtx.Current.ID, "rev", cmd.StateCtx.Current.Rev)
		if cmd.FlowID != `` {
			args = append(args, "to", cmd.FlowID)
		}
	case *ResumeCommand:
		args = append(args, "cmd", "resume", "id", cmd.StateCtx.Current.ID, "rev", cmd.StateCtx.Current.Rev)
	case *EndCommand:
		args = append(args, "cmd", "end", "id", cmd.StateCtx.Current.ID, "rev", cmd.StateCtx.Current.Rev)
	case *DelayCommand:
		args = append(args, "cmd", "delay", "id", cmd.StateCtx.Current.ID, "rev", cmd.StateCtx.Current.Rev, "dur", cmd.Duration)
	case *ExecuteCommand:
		args = append(args, "cmd", "execute", "id", cmd.StateCtx.Current.ID, "rev", cmd.StateCtx.Current.Rev)
	case *NoopCommand:
		args = append(args, "cmd", "noop", "id", cmd.StateCtx.Current.ID, "rev", cmd.StateCtx.Current.Rev)
	case *ReferenceDataCommand:
		args = append(args,
			"cmd", "reference_data",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
			"data_id", cmd.Data.ID,
			"data_rev", cmd.Data.Rev,
			"annot", cmd.Annotation,
		)
	case *DereferenceDataCommand:
		args = append(args,
			"cmd", "dereference_data",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
			"data_id", cmd.Data.ID,
			"data_rev", cmd.Data.Rev,
			"annot", cmd.Annotation,
		)
	case *StoreDataCommand:
		args = append(args, "cmd", "store_data", "data_id", cmd.Data.ID, "data_rev", cmd.Data.Rev)
	case *GetDataCommand:
		args = append(args, "cmd", "get_data", "data_id", cmd.Data.ID, "data_rev", cmd.Data.Rev)
	case *DeserializeCommand:
		args = append(args, "cmd", "deserialize")
	case *SerializeCommand:
		args = append(args, "cmd", "serialize")
	case *GetFlowCommand:
		args = append(args, "cmd", "get_flow", "flow_id", cmd.StateCtx.Current.Transition.ToID)
	case *WatchCommand:
		args = append(args, "cmd", "watch")
	default:
		args = append(args, "cmd", fmt.Sprintf("%T", cmd))
	}

	l.Info("do commands", args...)

}
