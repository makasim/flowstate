package stddoer

import "github.com/makasim/flowstate"

func Noop() flowstate.Doer {
	return flowstate.DoerFunc(func(cmd0 flowstate.Command) error {
		if _, ok := cmd0.(*flowstate.NoopCommand); !ok {
			return flowstate.ErrCommandNotSupported
		}

		return nil
	})
}
