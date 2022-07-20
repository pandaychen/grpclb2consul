package consul_discovery

//CONSUL watcher

import (
	"context"
	"fmt"

	//"google.golang.org/grpc/grpclog"
	"sync"

	consulapi "github.com/hashicorp/consul/api"
	consulwatcher "github.com/hashicorp/consul/api/watch"
	"go.uber.org/zap"
	"google.golang.org/grpc/resolver"
)

const CHANNEL_SIZE = 64

type ConsulWatcher struct {
	ServiceName     string
	ConsulConf      *consulapi.Config //CONSUL agent address
	ConsulWplan     *consulwatcher.Plan
	Ctx             context.Context
	Cancel          context.CancelFunc
	SyncWg          sync.WaitGroup
	ResovleAddrsOld []resolver.Address
	AddrsChannel    chan []resolver.Address
	Logger          *zap.Logger
	sync.RWMutex
}

func NewConsulWatcher(iconf *consulapi.Config, serviceName string, zlogger *zap.Logger) *ConsulWatcher {
	watcherplan, err := consulwatcher.Parse(map[string]interface{}{
		"type":    "service",
		"service": serviceName,
	})

	if err != nil {
		return nil
	}

	w := &ConsulWatcher{
		ServiceName:  serviceName,
		ConsulWplan:  watcherplan,
		ConsulConf:   iconf,
		AddrsChannel: make(chan []resolver.Address, CHANNEL_SIZE), //创建notify channel
		Logger:       zlogger,
	}

	//实现consul-watch的逻辑
	watcherplan.Handler = w.WatcherHandler

	return w
}

func (w *ConsulWatcher) Close() {
	defer w.SyncWg.Wait()
	w.ConsulWplan.Stop()
}

func (w *ConsulWatcher) Watch() chan []resolver.Address {
	go w.ConsulWplan.RunWithConfig(w.ConsulConf.Address, w.ConsulConf)
	return w.AddrsChannel
}

//传递给workplan的函数
func (w *ConsulWatcher) WatcherHandler(index uint64, cbdata interface{}) {
	srventrie_list, ok := cbdata.([]*consulapi.ServiceEntry)
	if !ok {
		w.Logger.Error("Get watcher callback data error")
		return
	}
	newaddrslist := make([]resolver.Address, 0)

	//top-level
	for _, entry := range srventrie_list {
		for _, check := range entry.Checks {
			//check和entry都是从ServiceEntry中获取
			if check.ServiceID == entry.Service.ID {
				//指定serviceName下的判断
				if consulapi.HealthPassing == check.Status {
					w.Logger.Info("Get Server Node", zap.String("serip", entry.Service.Address), zap.Int("port", entry.Service.Port))
					addr := fmt.Sprintf("%s:%d", entry.Service.Address, entry.Service.Port)
					//server+port传递给Resovler
					newaddrslist = append(newaddrslist, resolver.Address{Addr: addr, Metadata: &entry.Node.Meta /*interface{}*/})
				}
				break
			} else {
				w.Logger.Error("Unknown Service ID:", zap.String("check.SrvID", check.ServiceID), zap.String("entry.Service.ID", entry.Service.ID))
			}
		}
	}

	//notify all alive server address
	if !isSameAddrs(w.ResovleAddrsOld, newaddrslist) {
		w.ResovleAddrsOld = newaddrslist
		w.AddrsChannel <- w.NotifyAddresses(w.ResovleAddrsOld)
	}
}

//not known
func (w *ConsulWatcher) NotifyAddresses(in []resolver.Address) []resolver.Address {
	out := make([]resolver.Address, len(in))
	for i := 0; i < len(in); i++ {
		out[i] = in[i]
	}
	return out
}

//high-performance
func isSameAddrs(addrs1, addrs2 []resolver.Address) bool {
	if len(addrs1) != len(addrs2) {
		return false
	}
	for _, addr1 := range addrs1 {
		found := false
		for _, addr2 := range addrs2 {
			if addr1.Addr == addr2.Addr {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
