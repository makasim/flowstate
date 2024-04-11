package stddoer

import "github.com/makasim/flowstate"

func Transit() flowstate.Doer {
	return flowstate.DoerFunc(func(cmd0 flowstate.Command) error {
		cmd, ok := cmd0.(*flowstate.TransitCommand)
		if !ok {
			return flowstate.ErrCommandNotSupported
		}

		cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

		nextTs := flowstate.Transition{
			FromID:      cmd.StateCtx.Current.Transition.ToID,
			ToID:        cmd.FlowID,
			Annotations: nil,
		}

		cmd.StateCtx.Current.Transition = nextTs

		return nil
	})
}
