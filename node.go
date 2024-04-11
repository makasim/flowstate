package flowstate

type NodeID string

type Node struct {
	ID NodeID `json:"id"`

	BehaviorID BehaviorID `json:"behavior_id"`
}
