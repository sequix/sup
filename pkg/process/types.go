package process

type Request struct {
}

type Response struct {
	Message string
	SupPid  int
}

const (
	ActionStart       = "start"
	ActionStartWait   = "start-wait"
	ActionStop        = "stop"
	ActionStopWait    = "stop-wait"
	ActionRestart     = "restart"
	ActionRestartWait = "restart-wait"
	ActionKill        = "kill"
	ActionReload      = "reload"
	ActionStatus      = "status"
	ActionExit        = "exit"
	ActionExitWait    = "exit-wait"
)
