package daemon

//
//import (
//	"github.com/sequix/sup/pkg/config"
//	"github.com/sequix/sup/pkg/log"
//	"github.com/sequix/sup/pkg/meta"
//	"github.com/sequix/sup/pkg/usock"
//	"github.com/sequix/sup/pkg/util"
//)
//
//var (
//	userver *usock.Server
//)
//
//func Init() {
//	var err error
//	userver, err = usock.NewServer(config.G.SupConfig.Socket)
//	if err != nil {
//		log.Fatal("start userver: %s", err)
//	}
//}
//
//func Run(stopCh util.BroadcastCh) {
//	reqCh := userver.RequestCh()
//	for {
//		select {
//		case <-stopCh:
//			return
//		case reqWrap := <-reqCh:
//			req := &meta.Request{}
//			if err := reqWrap.DecodeInto(req); err != nil {
//				log.Error("decode req: %s", err)
//				continue
//			}
//			switch req.Action {
//			case meta.ActionStart:
//				start()
//			case meta.ActionRestart:
//				restart()
//			case meta.ActionStop:
//				stop()
//			case meta.ActionKill:
//				kill()
//			case meta.ActionReload:
//				reload()
//			case meta.ActionStatus:
//			default:
//				log.Error("unknown action %q", req.Action)
//			}
//		}
//	}
//}
//
//func start()   {}
//func stop()    {}
//func restart() {}
//func reload()  {}
//func kill()    {}
