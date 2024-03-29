# grpclb2consul
一个 gRPC 的负载均衡解析器实现 (基于 Consul)

## 说明

关于 GRPC-LB 的实现，可以参考下比较基础的文章：</br>
[GRPC 服务发现 & 负载均衡](https://segmentfault.com/a/1190000008672912) </br>
[gRPC Load Balancing](https://grpc.io/blog/loadbalancing/) </br>

本项目借助于 gRPC+Consul 实现了基础的服务注册与服务发现, 实现了如下两种调用方式：</br>

-	resolver 包, 借助于 GRPC 的 resolver/balancer 包提供的接口, 支持自定义的负载均衡算法 </br>
-	naming 包, 借助于 naming 包的 `Next()` 方法, 只能实现 gRPC 默认的 round-robin 方式 </br>

不管采用哪种方式, 其本质都是实现地址解析和更新策略 (gRPC 默认提供了 DNS 方式), 两种方式实现的思路如下:

####  naming 包
通过实现 `naming.Resolver` 和 `naming.Watcher` 接口来支持
- `naming.Resolver`：实现地址解析
- `naming.Watcher`：实现节点的变更, 添加 / 删除

## 服务注册
grpc-resovler 包的 Address 结构, 注意看其中的 Metadata, 可以在其中放入一些与 LB 特性相关的数据 (如权重等), 用于我们实现 LB 算法
```golang
type Address struct {
    // Addr is the server address on which a connection will be established.
    Addr string
    // Type is the type of this address.
    Type AddressType
    // ServerName is the name of this address.
    //
    // e.g. if Type is GRPCLB, ServerName should be the name of the remote load
    // balancer, not the name of the backend.
    ServerName string
    // Metadata is the information associated with Addr, which may be used
    // to make load balancing decision.
    Metadata interface{}
}
Address represents a server the client connects to. This is the EXPERIMENTAL API and may be changed or extended in the future.
```


## 服务发现

####  resolver
resolver[文档](https://pkg.go.dev/google.golang.org/grpc/resolver)：

```golang
type State struct {
	// Addresses is the latest set of resolved addresses for the target.
	Addresses []Address //address 为数组

	// ServiceConfig contains the result from parsing the latest service
	// config.  If it is nil, it indicates no service config is present or the
	// resolver does not provide service configs.
	ServiceConfig *serviceconfig.ParseResult

	// Attributes contains arbitrary data about the resolver intended for
	// consumption by the load balancing policy.
	Attributes *attributes.Attributes
}
```

## 负载均衡算法

- 带权重的 roundrobin 算法

- random 算法

- ketama 算法

- P2C 算法


## 服务访问

##	健康检查
健康检查在 Consul 实现服务发现中十分重要，Agent 会根据健康检查中设定的方法去检查服务的存活情况，一旦健康检查不通过，Consul 就会把此服务标识为不可访问（需要在 Resovler 中处理变化）。本例中实现了三种健康检查的方式：<br><br>
1.	TTL（TimeToLive）方式 <br>
该方式有点类似于 Etcd 的租约方式，应用服务需要自行实现定时上报心跳（TTL）的逻辑 <br>
2.	HTTP 方式 <br>
该方式需要在服务中新启动一个 http 服务（一般新启动一个 routine 来完成），Agent 定时向该接口发起 HTTP 请求，来完成服务的健康检查 <br>
3.	RPC 方式 <br>
该方式需要 RPC 服务中，实现标准的 GRPC 健康检查的方法 [GRPC-Health-check](https://github.com/grpc/grpc/blob/master/doc/health-checking.md) ，如下面的 Check 和 Wwatch 两个方法：</br>

```proto
syntax = "proto3";

package grpc.health.v1;

message HealthCheckRequest {
  string service = 1;
}

message HealthCheckResponse {
  enum ServingStatus {
    UNKNOWN = 0;
    SERVING = 1;
    NOT_SERVING = 2;
  }
  ServingStatus status = 1;
}

service Health {
  rpc Check(HealthCheckRequest) returns (HealthCheckResponse);

  rpc Watch(HealthCheckRequest) returns (stream HealthCheckResponse);
}
```


## 测试
1.	安装 consul, 开启 consul 的单机调试模式 <br>
```bash
consul agent -dev
==> Starting Consul agent...
==> Consul agent running!
           Version: 'v1.2.2'
           Node ID: 'fd735e74-9c6f-523d-04e8-fc7a4c9a46c7'
         Node name: 'VM_0_7_centos'
        Datacenter: 'dc1' (Segment: '<all>')
            Server: true (Bootstrap: false)
       Client Addr: [127.0.0.1] (HTTP: 8500, HTTPS: -1, DNS: 8600)
      Cluster Addr: 127.0.0.1 (LAN: 8301, WAN: 8302)
           Encrypt: Gossip: false, TLS-Outgoing: false, TLS-Incoming: false
(以下省略)
```

2、注册 / 查询服务 <br>
使用 `curl http://127.0.01:8500/v1/agent/services` 查询服务
```json
{
    "helloconsul-127.0.0.1-8001": {
        "Kind": "",
        "ID": "helloconsul-127.0.0.1-8001",
        "Service": "helloconsul",
        "Tags": [
            "{\"UniqID\":\"helloconsul-127.0.0.1-8001\",\"ServiceName\":\"helloconsul\",\"Ttl\":20,\"Ip\":\"127.0.0.1\",\"Port\":8001,\"Version\":\"v1.0\",\"HostName\":\"VM_0_7_centos\",\"Weight\":1,\"Metadata\":{\"othermsg\":\"none\"}}"
        ],
        "Meta": {},
        "Port": 8001,
        "Address": "127.0.0.1",
        "EnableTagOverride": false,
        "CreateIndex": 0,
        "ModifyIndex": 0,
        "ProxyDestination": "",
        "Connect": null
    }
}
```

## discuss

- ringbuffer@126.com
