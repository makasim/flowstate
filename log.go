package flowstate

import (
	"fmt"
	"log/slog"
	"strconv"
)

func logExecute(stateCtx *StateCtx, l *slog.Logger) {
	args := []any{
		"sess", stateCtx.sessID,
		"flow", stateCtx.Current.Transition.To,
		"id", stateCtx.Current.ID,
		"rev", stateCtx.Current.Rev,
	}

	currTs := stateCtx.Current.Transition

	if currTs.Annotations[StateAnnotation] != `` {
		args = append(args, "state", currTs.Annotations[StateAnnotation])
	}

	if currTs.Annotations[DelayUntilAnnotation] != `` {
		args = append(args, "delayed", "true")
	}

	args = append(args, "labels", stateCtx.Current.Labels)

	l.Info("engine: execute", args...)
}

func logDo(execSessID int64, cmd0 Command, subCmd bool, l *slog.Logger) {
	var args []any

	if execSessID > 0 {
		args = []any{"sess", strconv.FormatInt(execSessID, 10) + ":" + strconv.FormatInt(cmd0.SessID(), 10)}
	} else {
		args = []any{"sess", cmd0.SessID()}
	}

	cmdArg := "cmd"
	if subCmd {
		cmdArg = "sub_cmd"
	}

	switch cmd := cmd0.(type) {
	case *CommitCommand:
		args = append(args, cmdArg, "commit", "len", len(cmd.Commands))
	case *CommitStateCtxCommand:
		args = append(args, cmdArg, "commit_state_ctx", "id", cmd.StateCtx.Current.ID, "rev", cmd.StateCtx.Current.Rev)
	case *TransitCommand:
		args = append(args, cmdArg, "transit", "id", cmd.StateCtx.Current.ID, "rev", cmd.StateCtx.Current.Rev, "to", cmd.To)

		if len(cmd.StateCtx.Current.Labels) > 0 {
			args = append(args, "labels", cmd.StateCtx.Current.Labels)
		}
	case *PauseCommand:
		args = append(args,
			cmdArg, "pause",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
		)
		if cmd.To != `` {
			args = append(args, "to", cmd.To)
		}
		args = append(args, "labels", cmd.StateCtx.Current.Labels)
	case *ResumeCommand:
		args = append(args,
			cmdArg, "resume",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
		)
		if len(cmd.StateCtx.Current.Labels) > 0 {
			args = append(args, "labels", cmd.StateCtx.Current.Labels)
		}
	case *EndCommand:
		args = append(args,
			cmdArg, "end",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
		)
		if len(cmd.StateCtx.Current.Labels) > 0 {
			args = append(args, "labels", cmd.StateCtx.Current.Labels)
		}
	case *DelayCommand:
		args = append(args,
			cmdArg, "delay",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
			"execute_at", cmd.ExecuteAt,
		)
		if len(cmd.StateCtx.Current.Labels) > 0 {
			args = append(args, "labels", cmd.StateCtx.Current.Labels)
		}
	case *ExecuteCommand:
		args = append(args,
			cmdArg, "execute",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
		)
		if len(cmd.StateCtx.Current.Labels) > 0 {
			args = append(args, "labels", cmd.StateCtx.Current.Labels)
		}
	case *NoopCommand:
		args = append(args,
			cmdArg, "noop",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
		)
		if len(cmd.StateCtx.Current.Labels) > 0 {
			args = append(args, "labels", cmd.StateCtx.Current.Labels)
		}
	case *AttachDataCommand:
		args = append(args,
			cmdArg, "attach_data",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
			"data_id", cmd.Data.ID,
			"data_rev", cmd.Data.Rev,
			"alias", cmd.Alias,
		)
	case *GetDataCommand:
		args = append(args,
			cmdArg, "get_data",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
			"data_id", cmd.Data.ID,
			"data_rev", cmd.Data.Rev,
			"alias", cmd.Alias,
		)
	case *UnstackCommand:
		args = append(args, cmdArg, "unstack")
	case *StackCommand:
		args = append(args, cmdArg, "stack")
	case *GetStateByIDCommand:
		args = append(args, cmdArg, "get_state_by_id",
			"id", cmd.ID,
			"rev", cmd.Rev,
		)
	case *GetStateByLabelsCommand:
		args = append(args, cmdArg, "get_state_by_labels",
			"labels", cmd.Labels,
		)
	case *GetStatesCommand:
		args = append(args, cmdArg, "get_states")
		if cmd.SinceRev > 0 {
			args = append(args, "since_rev", cmd.SinceRev)
		}
		if cmd.LatestOnly {
			args = append(args, "latest_only", true)
		}
		if !cmd.SinceTime.IsZero() {
			args = append(args, "since_time", cmd.SinceTime)
		}
		for i, labels := range cmd.Labels {
			args = append(args, fmt.Sprintf("labels[%d]", i), labels)
		}
		if cmd.Limit != GetStatesDefaultLimit {
			args = append(args, "limit", cmd.Limit)
		}
	case *GetDelayedStatesCommand:
		args = append(args, cmdArg, "get_delayed_states",
			"since", cmd.Since,
			"until", cmd.Until,
			"offset", cmd.Offset,
		)
	default:
		args = append(args, cmdArg, fmt.Sprintf("%T", cmd))
	}

	l.Info("engine: do", args...)

}
