# grpclb2consul 
一个grpc的负载均衡解析器实现(基于CONSUL)

# 说明

关于GRPC-LB的实现，可以参考下比较基础的文章：</br>
[GRPC服务发现&负载均衡](https://segmentfault.com/a/1190000008672912) </br>
[gRPC Load Balancing](https://grpc.io/blog/loadbalancing/) </br>

本项目借助于GRPC+CONSUL实现了基础的服务注册与服务发现,实现了如下两种调用方式：</br>

-	resolver包,借助于GRPC的resolver/balancer包提供的接口,支持自定义的负载均衡算法</br>
-	naming包,借助于naming包的Next()方法,只能实现GRPC默认的round-robin方式</br>

不管采用哪种方式,其本质都是实现地址解析和更新策略(GRPC默认提供了DNS方式),两种方式实现的思路如下:

-	naming包
通过实现 naming.Resolver 和 naming.Watcher 接口来支持
```
naming.Resolver: 实现地址解析
naming.Watcher: 实现节点的变更,添加/删除
```

## 服务注册
grpc-resovler包的Address结构,注意看其中的Metadata,可以在其中放入一些与LB特性相关的数据(如权重等),用于我们实现LB算法
```
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


## 负载均衡算法

- 带权重的roundrobin算法 

- random算法

- ketama算法

- P2C算法


## 服务访问

##	健康检查
健康检查在Consul实现服务发现中十分重要，Agent会根据健康检查中设定的方法去检查服务的存活情况，一旦健康检查不通过，Consul就会把此服务标识为不可访问（需要在Resovler中处理变化）。本例中实现了三种健康检查的方式：<br><br>
1.	TTL（TimeToLive）方式<br>
该方式有点类似于Etcd的租约方式，应用服务需要自行实现定时上报心跳（TTL）的逻辑<br>
2.	HTTP方式<br>
该方式需要在服务中新启动一个http服务（一般新启动一个routine来完成），Agent定时向该接口发起HTTP请求，来完成服务的健康检查<br>
3.	RPC方式<br>
该方式需要RPC服务中，实现标准的GRPC健康检查的方法[GRPC-Health-check](https://github.com/grpc/grpc/blob/master/doc/health-checking.md) ，如下面的Check和Wwatch两个方法：</br>
```
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
1.	安装consul,开启consul的单机调试模式
```
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

- 注册服务

```
使用curl http://127.0.01:8500/v1/agent/services 查询服务

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
