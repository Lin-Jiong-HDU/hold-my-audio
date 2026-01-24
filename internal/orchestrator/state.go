package orchestrator

// State represents the current state of the orchestrator
type State int

const (
	IDLE State = iota
	PLAYING
	INTERRUPTED
	THINKING
	UPDATING
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case IDLE:
		return "IDLE"
	case PLAYING:
		return "PLAYING"
	case INTERRUPTED:
		return "INTERRUPTED"
	case THINKING:
		return "THINKING"
	case UPDATING:
		return "UPDATING"
	default:
		return "UNKNOWN"
	}
}
