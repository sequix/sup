package process

type Request struct {
}

type Response struct {
	Message string
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
)
