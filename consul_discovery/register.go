package consul_discovery

//consulapi for registry

import (
	"context"
	"encoding/json"
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
	"go.uber.org/zap"

	"time"
)

var (
	DefaultDeregisterCriticalServiceAfter = "1m"
	DefaultCheckInterval                  = "3s"
)

//register common node info
type GenericServerNodeValue struct {
	UniqID      string //唯一ID(CONSUL服务标识)
	ServiceName string // 服务名称(CONSUL服务识别)

	//上面这两个字段非常重要
	Ttl      int               //ttl seconds
	Ip       string            // 服务IP
	Port     int               //服务PORT
	Version  string            // 服务版本号，用于服务升级过程中，配置兼容问题
	HostName string            // 主机名称
	Weight   int               // 服务权重
	Metadata map[string]string //服务端与客户端可以约定相关格式
	//Metadata metadata.MD //使用通用结构
}

type ConsulRegistry struct {
	Ctx            context.Context //background()
	Cancel         context.CancelFunc
	ConsulConf     *consulapi.Config
	ConsulAgent    *consulapi.Client //consul-agent
	HeadlthCheckId string
	//UniqID       string //注册在consul中的唯一ID
	//ServiceName  string
	//Ttl          int //ttl seconds
	Logger       *zap.Logger
	GeneNodeData GenericServerNodeValue
}

//USER-DEFINE-CONFIG
type InitConfig struct {
	ConsulCfg *consulapi.Config
	Logger    *zap.Logger
}

func NewConsulRegistry(iconf *InitConfig, gnode *GenericServerNodeValue) (*ConsulRegistry, error) {
	client, err := consulapi.NewClient(iconf.ConsulCfg)
	if err != nil {
		iconf.Logger.Error("Create consul agent error", zap.String("errmsg", err.Error()))
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	//checkid := fmt.Sprint("%s-%s", gnode.ServiceName, gnode.UniqID)
	checkid := fmt.Sprintf("service:%s", gnode.UniqID) //checkid必须无非法字符,且为固定格式
	consul_registry := ConsulRegistry{
		Ctx:            ctx, //background()
		Cancel:         cancel,
		ConsulAgent:    client,
		ConsulConf:     iconf.ConsulCfg,
		HeadlthCheckId: checkid,
		//UniqID:      gnode.UniqID,
		//ServiceName: gnode.ServiceName,
		//Ttl:         gnode.Ttl,
		Logger:       iconf.Logger,
		GeneNodeData: *gnode,
	}

	return &consul_registry, nil
}

// 通过TTL-CHECK注册
func (c *ConsulRegistry) RegisterWithHealthCheckTTL() error {
	metadata, err := json.Marshal(c.GeneNodeData)
	if err != nil {
		c.Logger.Error("JSON marshal error", zap.String("errmsg", err.Error()))
		return err
	}
	tags := make([]string, 0)
	tags = append(tags, string(metadata))

	registerfunc := func() error {
		healthcheck := &consulapi.AgentServiceCheck{
			TTL:                            fmt.Sprintf("%ds", c.GeneNodeData.Ttl),
			Status:                         consulapi.HealthPassing,
			DeregisterCriticalServiceAfter: DefaultDeregisterCriticalServiceAfter,
		}
		fmt.Println(c.GeneNodeData.ServiceName)
		crs := &consulapi.AgentServiceRegistration{
			ID:      c.GeneNodeData.UniqID, //uniq-id
			Name:    c.GeneNodeData.ServiceName,
			Address: c.GeneNodeData.Ip,   // 服务 IP
			Port:    c.GeneNodeData.Port, // 服务端口
			Tags:    tags,                // tags，可以为空([]string{})
			Check:   healthcheck}
		err := c.ConsulAgent.Agent().ServiceRegister(crs) //单例模式
		if err != nil {
			c.Logger.Error("Register with consul error", zap.String("errmsg", err.Error()))
			return fmt.Errorf("Register with consul error: %s\n", err.Error())
		}
		return nil
	}

	err = registerfunc()
	if err != nil {
		c.Logger.Error("Register with consul error", zap.String("errmsg", err.Error()))
		return err
	}

	//TTL-续期
	TTLTicker := time.NewTicker(time.Duration(c.GeneNodeData.Ttl) * time.Second / 2)
	RenewRegisterTicker := time.NewTicker(time.Minute)

	for {
		select {
		case <-c.Ctx.Done():
			TTLTicker.Stop()
			RenewRegisterTicker.Stop()
			c.ConsulAgent.Agent().ServiceDeregister(c.GeneNodeData.UniqID) //cancel service
			return nil
		case <-TTLTicker.C:
			fmt.Println("checkid:", c.HeadlthCheckId)
			c.Logger.Error("Register with consul TTL", zap.String("errmsg", c.HeadlthCheckId))
			err := c.ConsulAgent.Agent().PassTTL(c.HeadlthCheckId, "")
			if err != nil {
				c.Logger.Error("Register with consul TTL(health-check) error", zap.String("errmsg", err.Error()))
			}
		case <-RenewRegisterTicker.C:
			err = registerfunc() //因为这里采用了定时上报的方式，所以health check中设置的是TTL模式，除了TTL，还有tcp-check，grpc-check等方式
			if err != nil {
				c.Logger.Error("Renew Register with consul error", zap.String("errmsg", err.Error()))
			}
		}
	}

	return nil
}

//使用GRPC健康检查方式注册
func (c *ConsulRegistry) RegisterWithHealthCheckGRPC() error {
	var (
		err error
	)
	tags := []string{"tag1", "tag2"}

	registerfunc := func() error {
		//健康检查
		healthcheck := &consulapi.AgentServiceCheck{
			Interval:                       DefaultCheckInterval,                                                                        // 健康检查间隔                                            // 健康检查间隔
			GRPC:                           fmt.Sprintf("%s:%d/%s", c.GeneNodeData.Ip, c.GeneNodeData.Port, c.GeneNodeData.ServiceName), // grpc 支持，执行健康检查的地址，service 会传到 Health.Check 函数中
			DeregisterCriticalServiceAfter: DefaultDeregisterCriticalServiceAfter,                                                       // 注销时间，相当于过期时间
		}

		crs := &consulapi.AgentServiceRegistration{
			ID:      c.GeneNodeData.UniqID, //uniq-id
			Name:    c.GeneNodeData.ServiceName,
			Address: c.GeneNodeData.Ip,   // 服务 IP
			Port:    c.GeneNodeData.Port, // 服务端口
			Tags:    tags,                // tags，可以为空([]string{})
			Meta:    c.GeneNodeData.Metadata,
			Check:   healthcheck}
		err := c.ConsulAgent.Agent().ServiceRegister(crs) //单例模式
		if err != nil {
			c.Logger.Error("Register with consul error", zap.String("errmsg", err.Error()))
			return fmt.Errorf("Register with consul error: %s\n", err.Error())
		}
		return nil
	}

	err = registerfunc()
	if err != nil {
		c.Logger.Error("Register with consul error", zap.String("errmsg", err.Error()))
		return err
	}

	//TTL-续期
	TTLTicker := time.NewTicker(time.Duration(c.GeneNodeData.Ttl) * time.Second / 5)
	RenewRegisterTicker := time.NewTicker(time.Minute)

	for {
		select {
		case <-c.Ctx.Done():
			TTLTicker.Stop()
			RenewRegisterTicker.Stop()
			c.ConsulAgent.Agent().ServiceDeregister(c.GeneNodeData.UniqID) //cancel service
			return nil
		case <-RenewRegisterTicker.C:
			err = registerfunc() //因为这里采用了定时上报的方式，所以health check中设置的是TTL模式，除了TTL，还有tcp-check，grpc-check等方式
			if err != nil {
				c.Logger.Error("Renew Register with consul error", zap.String("errmsg", err.Error()))
			}
		}
	}

	return nil
}

func (c *ConsulRegistry) Unregister() error {
	c.Cancel()
	return nil
}
