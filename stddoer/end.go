package stddoer

import "github.com/makasim/flowstate"

func End() flowstate.Doer {
	return flowstate.DoerFunc(func(cmd0 flowstate.Command) error {
		cmd, ok := cmd0.(*flowstate.EndCommand)
		if !ok {
			return flowstate.ErrCommandNotSupported
		}

		cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

		nextTs := flowstate.Transition{
			FromID:      cmd.StateCtx.Current.Transition.ToID,
			ToID:        ``,
			Annotations: nil,
		}
		nextTs.SetAnnotation(flowstate.StateAnnotation, `ended`)

		cmd.StateCtx.Current.Transition = nextTs

		return nil
	})
}
