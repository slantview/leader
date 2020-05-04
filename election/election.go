package election

// LeaderNamespace is our service name for all backends.
const LeaderNamespace = "/service/election/leader"

// Election is the interface for our Leader Election backend.
type Election interface {
	Init() error
	IsLeader() bool
	GetLeader() string
}
