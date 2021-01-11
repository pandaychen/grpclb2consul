package main

//a simple grpc-client
//author:pandaychen

import (
	"fmt"
	"log"
	"os"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	consulresovler "github.com/pandaychen/grpclb2consul/consul_discovery"
	"github.com/pandaychen/grpclb2consul/enums"
	proto "github.com/pandaychen/grpclb2consul/proto"
	"github.com/pandaychen/grpclb2consul/utils"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"gopkg.in/urfave/cli.v1"
)

func RpcClientStart(consul_addrstr, servicename, lbtype string) {
	consulresovler.RegisterResolver(string(enums.RT_CONSUL), &consulapi.Config{Address: consul_addrstr}, servicename)
	c, err := grpc.Dial(utils.GetGrpcScheme(string(enums.RT_CONSUL)), grpc.WithInsecure(), grpc.WithBalancerName("round_robin"))
	if err != nil {
		log.Printf("grpc dial: %s", err)
		return
	}
	defer c.Close()

	client := proto.NewTestClient(c)
	for i := 0; i < 100000; i++ {

		resp, err := client.Say(context.Background(), &proto.SayReq{Content: "round robin"})
		if err != nil {
			log.Println(err)
			continue
		}
		fmt.Println(resp.Content)
		resp2, err := client.AddService(context.Background(), &proto.AddIntNumsRequest{A: int64(i), B: int64(i)})
		if err != nil {
			log.Println(err)
			continue
		}
		time.Sleep(time.Second)
		fmt.Println(resp2.Result, resp2.Err)
	}
}

func CmdRun() {
	var consul_addrstr string
	var servicename string
	var lbtype string

	app := cli.NewApp()
	app.Name = "Consul Grpc Client"
	app.Usage = ""
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "consul,cs",
			Value:       "http://127.0.0.1:8500",
			Usage:       "Consul Agent address list",
			Destination: &consul_addrstr,
		},
		cli.StringFlag{
			Name:        "service name,sn",
			Value:       "helloconsul",
			Usage:       "service name",
			Destination: &servicename,
		},
		cli.StringFlag{
			Name:        "balancer ,lb",
			Value:       "rr",
			Usage:       "load balancer[rr|random|consistent]",
			Destination: &lbtype,
		},
	}

	app.Action = func(c *cli.Context) error {
		RpcClientStart(consul_addrstr, servicename, lbtype)
		return nil
	}
	app.Before = func(c *cli.Context) error {
		return nil
	}
	app.After = func(c *cli.Context) error {
		return nil
	}

	cli.HelpFlag = cli.BoolFlag{
		Name:  "help, h",
		Usage: "Help!Help!",
	}

	cli.VersionFlag = cli.BoolFlag{
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
