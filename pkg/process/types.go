package process

type Request struct {
}

type Response struct {
	Message string
	SupPid  int
}

const (
	ActionStart   = "start"
	ActionStop    = "stop"
	ActionRestart = "restart"
	ActionKill    = "kill"
	ActionReload  = "reload"
	ActionStatus  = "status"
	ActionExit    = "exit"
)
