package main

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/examples"
)

func main() {
	slog.Default().Info("Example of state machine - money transfer")

	e, _, _, tearDown := examples.SetUp()
	defer tearDown()

	johnBalanceStateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: `john_balance`,
			Annotations: map[string]string{
				// the balance could be stored as json or any other format in attachable data
				`balance`: `100`,
			},
		},
	}

	sarahBalanceStateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: `sarah_balance`,
			Annotations: map[string]string{
				`balance`: `50`,
			},
		},
	}

	// commit initial balances
	err := e.Do(flowstate.Commit(
		flowstate.Park(johnBalanceStateCtx),
		flowstate.Park(sarahBalanceStateCtx),
	))
	examples.HandleError(err)

	// simulate John updated his balance with 50, but we are not aware of it yet.
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)

		// get the latest state of John's balance
		johnBalanceStateCtx := &flowstate.StateCtx{}
		err := e.Do(flowstate.GetStateByID(johnBalanceStateCtx, `john_balance`, 0))
		examples.HandleError(err)

		setBalance(johnBalanceStateCtx, getBalance(johnBalanceStateCtx)+50)

		// commit the updated balance
		err = e.Do(flowstate.Commit(flowstate.Park(johnBalanceStateCtx)))
		examples.HandleError(err)

		slog.Default().Info(fmt.Sprintf("John's balance has been updated to %s", johnBalanceStateCtx.Current.Annotations[`balance`]))
	}()

	<-doneCh

	for {
		// now the balance has been updated, let's try transfer 30 from John to Sarah
		johnBalance := getBalance(johnBalanceStateCtx)
		sarahBalance := getBalance(sarahBalanceStateCtx)

		setBalance(johnBalanceStateCtx, johnBalance-30)
		setBalance(sarahBalanceStateCtx, sarahBalance+30)

		// transfer commit should fail because John's balance has been updated
		// we should get a revision mismatch error
		err = e.Do(flowstate.Commit(
			flowstate.Park(johnBalanceStateCtx),
			flowstate.Park(sarahBalanceStateCtx),
		))
		if flowstate.IsErrRevMismatch(err) {
			slog.Default().Info("The balances are not up to date, getting the latest states...")

			// let's try to get the latest state of John's and Sarah balances, and try again
			johnBalanceStateCtx = &flowstate.StateCtx{}
			sarahBalanceStateCtx = &flowstate.StateCtx{}
			err := e.Do(
				flowstate.GetStateByID(johnBalanceStateCtx, `john_balance`, 0),
				flowstate.GetStateByID(sarahBalanceStateCtx, `sarah_balance`, 0),
			)
			examples.HandleError(err)

			slog.Default().Info(fmt.Sprintf("John's balance is %s, Sarah's balance is %s",
				johnBalanceStateCtx.Current.Annotations[`balance`],
				sarahBalanceStateCtx.Current.Annotations[`balance`],
			))

			continue
		} else if err != nil {
			examples.HandleError(err)
		}

		// the transfer succeeded
		slog.Default().Info("The transfer succeeded.")
		slog.Default().Info(fmt.Sprintf("John's balance is %s, Sarah's balance is %s",
			johnBalanceStateCtx.Current.Annotations[`balance`],
			sarahBalanceStateCtx.Current.Annotations[`balance`],
		))

		break
	}

}

func getBalance(stateCtx *flowstate.StateCtx) int {
	balance, err := strconv.Atoi(stateCtx.Current.Annotations[`balance`])
	examples.HandleError(err)

	return balance
}

func setBalance(stateCtx *flowstate.StateCtx, balance int) {
	stateCtx.Current.Annotations[`balance`] = strconv.Itoa(balance)
}
