package flowstate

type TransitionID string

type Transition struct {
	ID     TransitionID `json:"id"`
	FromID NodeID       `json:"from"`
	ToID   NodeID       `json:"to"`
}
