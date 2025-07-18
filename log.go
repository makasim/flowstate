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

func LogCommand(msg string, cmd0 Command, l *slog.Logger) {
	logCommand(msg, 0, cmd0, l)
}

func logCommand(msg string, execSessID int64, cmd0 Command, l *slog.Logger) {
	var args []any

	if execSessID > 0 {
		args = []any{"sess", strconv.FormatInt(execSessID, 10) + ":" + strconv.FormatInt(cmd0.SessID(), 10)}
	} else {
		args = []any{"sess", cmd0.SessID()}
	}

	switch cmd := cmd0.(type) {
	case *CommitCommand:
		args = append(args, "cmd", "commit", "len", len(cmd.Commands))
		l.Info(msg, args...)

		for _, subCmd := range cmd.Commands {
			logCommand("commit sub cmd", execSessID, subCmd, l)
		}
		return
	case *CommitStateCtxCommand:
		args = append(args, "cmd", "commit_state_ctx", "id", cmd.StateCtx.Current.ID, "rev", cmd.StateCtx.Current.Rev)
	case *TransitCommand:
		args = append(args, "cmd", "transit", "id", cmd.StateCtx.Current.ID, "rev", cmd.StateCtx.Current.Rev, "to", cmd.To)

		if len(cmd.StateCtx.Current.Labels) > 0 {
			args = append(args, "labels", cmd.StateCtx.Current.Labels)
		}
	case *PauseCommand:
		args = append(args,
			"cmd", "pause",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
		)
		if cmd.To != `` {
			args = append(args, "to", cmd.To)
		}
		args = append(args, "labels", cmd.StateCtx.Current.Labels)
	case *ResumeCommand:
		args = append(args,
			"cmd", "resume",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
		)
		if len(cmd.StateCtx.Current.Labels) > 0 {
			args = append(args, "labels", cmd.StateCtx.Current.Labels)
		}
	case *EndCommand:
		args = append(args,
			"cmd", "end",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
		)
		if len(cmd.StateCtx.Current.Labels) > 0 {
			args = append(args, "labels", cmd.StateCtx.Current.Labels)
		}
	case *DelayCommand:
		args = append(args,
			"cmd", "delay",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
			"execute_at", cmd.ExecuteAt,
		)
		if len(cmd.StateCtx.Current.Labels) > 0 {
			args = append(args, "labels", cmd.StateCtx.Current.Labels)
		}
	case *ExecuteCommand:
		args = append(args,
			"cmd", "execute",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
		)
		if len(cmd.StateCtx.Current.Labels) > 0 {
			args = append(args, "labels", cmd.StateCtx.Current.Labels)
		}
	case *NoopCommand:
		args = append(args,
			"cmd", "noop",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
		)
		if len(cmd.StateCtx.Current.Labels) > 0 {
			args = append(args, "labels", cmd.StateCtx.Current.Labels)
		}
	case *AttachDataCommand:
		args = append(args,
			"cmd", "attach_data",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
			"data_id", cmd.Data.ID,
			"data_rev", cmd.Data.Rev,
			"alias", cmd.Alias,
		)
	case *GetDataCommand:
		args = append(args,
			"cmd", "get_data",
			"id", cmd.StateCtx.Current.ID,
			"rev", cmd.StateCtx.Current.Rev,
			"data_id", cmd.Data.ID,
			"data_rev", cmd.Data.Rev,
			"alias", cmd.Alias,
		)
	case *UnstackCommand:
		args = append(args, "cmd", "unstack")
	case *StackCommand:
		args = append(args, "cmd", "stack")
	case *GetStateByIDCommand:
		args = append(args, "cmd", "get_state_by_id",
			"id", cmd.ID,
			"rev", cmd.Rev,
		)
	case *GetStateByLabelsCommand:
		args = append(args, "cmd", "get_state_by_labels",
			"labels", cmd.Labels,
		)
	case *GetStatesCommand:
		args = append(args, "cmd", "get_states")
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
		args = append(args, "cmd", "get_delayed_states",
			"since", cmd.Since,
			"until", cmd.Until,
			"offset", cmd.Offset,
		)
	default:
		args = append(args, "cmd", fmt.Sprintf("%T", cmd))
	}

	l.Info(msg, args...)
}
