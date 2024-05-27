package stddoer

import "github.com/makasim/flowstate"

func Pause() flowstate.Doer {
	return flowstate.DoerFunc(func(cmd0 flowstate.Command) error {
		cmd, ok := cmd0.(*flowstate.PauseCommand)
		if !ok {
			return flowstate.ErrCommandNotSupported
		}

		cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

		nextTs := flowstate.Transition{
			FromID:      cmd.StateCtx.Current.Transition.ToID,
			ToID:        cmd.FlowID,
			Annotations: nil,
		}
		nextTs.SetAnnotation(flowstate.StateAnnotation, `paused`)

		cmd.StateCtx.Current.Transition = nextTs

		return nil
	})
}
