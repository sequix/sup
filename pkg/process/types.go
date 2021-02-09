package process

type Request struct {
}

type Response struct {
	Message string
}

const (
	ActionStart   = "start"
	ActionStop    = "stop"
	ActionKill    = "kill"
	ActionRestart = "restart"
	ActionReload  = "reload"
	ActionStatus  = "status"
)
