package main

//a simple grpc-server,use healthy-check
//author:pandaychen

import (
	//"github.com/pandaychen/grpclb2consul/balancer"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	consulapi "github.com/hashicorp/consul/api"
	consulreg "github.com/pandaychen/grpclb2consul/consul_discovery"
	"github.com/pandaychen/grpclb2consul/enums"
	proto "github.com/pandaychen/grpclb2consul/proto"
	"github.com/pandaychen/grpclb2consul/utils"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	//"time"
	"errors"

	hc "github.com/pandaychen/grpclb2consul/healthcheck"
)

type ConsulServer struct {
	Addr        string
	Weight      int
	ConsulAgent string //consul agent addr
	Port        int
	NodeId      string
	Grpcsrv     *grpc.Server
	Logger      *zap.Logger
	ConsulReg   *consulreg.ConsulRegistry
	CheckType   string
	sync.WaitGroup
}

func NewConsulServer(reg *consulreg.ConsulRegistry, addr, consul_agent, nodeid string, port, weight int, checktype string) *ConsulServer {
	srv := grpc.NewServer()
	zlogger, _ := utils.ZapLoggerInit(string(enums.SERVER_ZLOG_NAME))
	cs := &ConsulServer{
		ConsulReg:   reg,
		Addr:        addr,
		Weight:      weight,
		Port:        port,
		ConsulAgent: consul_agent,
		NodeId:      nodeid,
		Grpcsrv:     srv,
		CheckType:   checktype,
		Logger:      zlogger}
	return cs
}

func (cs *ConsulServer) ServerRun() {
	listener, err := net.Listen("tcp", cs.Addr)
	if err != nil {
		cs.Logger.Error("failed to binding", zap.String("errmsg", err.Error()))
		return
	}
	cs.Logger.Info("rpc listening succ", zap.String("serveraddr", cs.Addr))

	//cs实现了Say方法,所以可以注册(protobuf的方法)
	proto.RegisterTestServer(cs.Grpcsrv, cs)
	//注册健康检查
	grpc_health_v1.RegisterHealthServer(cs.Grpcsrv, &hc.HealthyCheck{})

	cs.Grpcsrv.Serve(listener)
}

func (cs *ConsulServer) GracefulStop() {
	cs.Grpcsrv.GracefulStop()
}

// 业务逻辑
func (cs *ConsulServer) Say(ctx context.Context, req *proto.SayReq) (*proto.SayResp, error) {
	text := "Hello " + req.Content
	log.Println(text)
	return &proto.SayResp{Content: text}, nil
}

func (cs *ConsulServer) AddService(ctx context.Context, request *proto.AddIntNumsRequest) (*proto.AddIntNumsResponse, error) {
	/*
		if rand.Int()%2 == 0 {
			time.Sleep(time.Duration(200) * time.Millisecond)
		}
	*/
	response := &proto.AddIntNumsResponse{
		Result: request.A + request.B,
		Err:    "",
	}
	return response, nil
}

func (cs *ConsulServer) GoRegister() {
	cs.Add(1)
	defer cs.Done()
	go func() {
		if cs.CheckType == enums.CONSUL_HealthCheckType_TTL {
			cs.ConsulReg.RegisterWithHealthCheckTTL()
		} else if cs.CheckType == enums.CONSUL_HealthCheckType_RPC {
			cs.ConsulReg.RegisterWithHealthCheckGRPC()
		} else {
			panic(errors.New("Not support Health Check"))
		}
	}()
}

func (cs *ConsulServer) GoServer() {
	cs.Add(1)
	defer cs.Done()
	go func() {
		cs.ServerRun()
	}()
}

/////END of consul server wrapper////

func RpcServerStart(servernode, consul_addrstr, bind_addr string, port, weight int, servicename, checktype, server_version string) {
	consulconf := &consulapi.Config{
		Address: consul_addrstr,
	}
	zlogger, _ := utils.ZapLoggerInit(servicename)

	gnode := new(consulreg.GenericServerNodeValue)
	gnode.UniqID = fmt.Sprintf("%v-%v-%v", servicename, bind_addr, port) // 服务节点的名称
	//fmt.Println("id=", gnode.UniqID)
	gnode.ServiceName = servicename
	//fmt.Println(servicename)
	gnode.Ttl = 20
	gnode.Ip = bind_addr
	gnode.Port = port
	gnode.Version = server_version
	hostname, _ := os.Hostname()
	gnode.HostName = hostname
	gnode.Weight = weight
	gnode.Metadata = map[string]string{"othermsg": "none"}

	config := &consulreg.InitConfig{
		ConsulCfg: consulconf,
		Logger:    zlogger,
	}

	registrar, err := consulreg.NewConsulRegistry(config, gnode)
	if err != nil {
		panic(err)
		return
	}

	//generate grpc server
	s := NewConsulServer(registrar, fmt.Sprintf("%s:%d", bind_addr, port), consul_addrstr, servernode, port, weight, checktype)

	s.GoRegister()
	s.GoServer()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	s.ConsulReg.Unregister()
	s.GracefulStop()
	s.Wait()
}

func CmdRun() {
	var servernode string
	var consul_addrstr string
	var bind_addr string
	var servicename string
	var checktype string
	var server_version string

	app := cli.NewApp()
	app.Name = "Consul Grpc Server"
	app.Usage = "a simple grpc server,use consul as service registy"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		&cli.IntFlag{
			Name:  "port, p",
			Value: 8001,
			Usage: "listening port",
		},
		&cli.StringFlag{
			Name:        "bind,b",
			Value:       "127.0.0.1",
			Usage:       "Bind Addr",
			Destination: &bind_addr,
		},
		&cli.StringFlag{
			Name:        "nodeid,n",
			Value:       "snode0",
			Usage:       "Service name prefix to consul",
			Destination: &servernode,
		},
		&cli.StringFlag{
			Name:        "consul,cs",
			Value:       "http://127.0.0.1:8500",
			Usage:       "Consul Agent address list",
			Destination: &consul_addrstr,
		},
		&cli.StringFlag{
			Name:        "service name,sn",
			Value:       "helloconsul",
			Usage:       "service name",
			Destination: &servicename,
		},
		&cli.StringFlag{
			Name:        "check type,ct",
			Value:       "ttl",
			Usage:       "healthy check type[ttl|grpc|http]",
			Destination: &checktype,
		},
		&cli.StringFlag{
			Name:        "version,sv",
			Value:       "v1.0",
			Usage:       "service version",
			Destination: &server_version,
		},
		&cli.IntFlag{
			Name:  "weight, w",
			Value: 1,
			Usage: "service node weight",
		},
	}

	app.Action = func(c *cli.Context) error {
		//fmt.Println(c.String("nodeid"), c.Int("port"))
		//fmt.Println(servernode, consul_addrstr)
		RpcServerStart(servernode, consul_addrstr, bind_addr, c.Int("port"), c.Int("weight"), servicename, checktype, server_version)
		return nil
	}
	app.Before = func(c *cli.Context) error {
		return nil
	}
	app.After = func(c *cli.Context) error {
		return nil
	}

	cli.HelpFlag = &cli.BoolFlag{
		Name:  "help, h",
		Usage: "Help!Help!",
	}

	cli.VersionFlag = &cli.BoolFlag{
		Name:  "print-version, v",
		Usage: "print version",
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	CmdRun()
}
