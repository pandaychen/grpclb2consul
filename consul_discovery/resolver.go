package consul_discovery

//consul resovler(CLIENT)

import (
	"github.com/pandaychen/grpclb2consul/utils"
	consulapi "github.com/hashicorp/consul/api"
	"go.uber.org/zap"
	"google.golang.org/grpc/resolver"
	"sync"
	//"time"
)

type ConsulResolver struct {
	scheme      string            //for GRCP-CLIENT DIAL
	ServiceName string            //监控哪个service变化
	ConsulConf  *consulapi.Config //agent Address
	Watcher     *ConsulWatcher    //每个resovler包含一个watcher
	ClientConn  resolver.ClientConn
	SyncWg      sync.WaitGroup
	Logger      *zap.Logger
}

//RegisterResolver+Build ==>returns a resolver
func RegisterResolver(scheme string, consulConf *consulapi.Config, srvName string) {
	zlogger, _ := utils.ZapLoggerInit(srvName)
	consul_resovler := &ConsulResolver{
		scheme:      scheme,     //grpc-dial
		ServiceName: srvName,    //监听哪个service
		ConsulConf:  consulConf, //consul-agent配置
		Logger:      zlogger,
	}
	resolver.Register(consul_resovler)
}

//resovler build
func (r *ConsulResolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	//construct a resolver
	r.ClientConn = cc
	r.Watcher = NewConsulWatcher(r.ConsulConf, r.ServiceName, r.Logger)
	r.SyncWg.Add(1)
	go r.startRecvDynamicAddrlist()
	return r, nil
}

func (r *ConsulResolver) startRecvDynamicAddrlist() {
	defer r.SyncWg.Done()
	addrlist_chan := r.Watcher.Watch()
	for addr := range addrlist_chan {
		//addr is a slice,addrlist_chan is a slice channel
		//r.ClientConn.NewAddress(addr)
		r.ClientConn.UpdateState(resolver.State{Addresses: addr})
	}
}

func (r *ConsulResolver) ResolveNow(o resolver.ResolveNowOption) {
	r.Logger.Info("ResolveNow")
}

func (r *ConsulResolver) Close() {
	r.Watcher.Close()
	r.SyncWg.Wait()
}

//FOR DIAL
func (r *ConsulResolver) Scheme() string {
	return r.scheme
}
