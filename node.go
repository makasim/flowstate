package flowstate

// deprecated
type NodeID string

// deprecated
type Node struct {
	ID NodeID `json:"id"`

	BehaviorID BehaviorID `json:"behavior_id"`
}
