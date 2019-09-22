# grpclb2consul 
一个grpc的负载均衡解析器实现(基于CONSUL)

# 说明

关于GRPC-LB的实现，可以参考下比较基础的文章：</br>
[GRPC服务发现&负载均衡](https://segmentfault.com/a/1190000008672912) </br>
[gRPC Load Balancing](https://grpc.io/blog/loadbalancing/) </br>

本项目借助于GRPC+CONSUL实现了基础的服务注册与服务发现,实现了如下两种调用方式：</br>

-	resolver包,借助于GRPC的resolver/balancer包提供的接口,支持自定义的负载均衡算法</br>
-	naming包,借助于naming包的Next()方法,只能实现GRPC默认的round-robin方式</br>

具体实现的思路如下:

## 服务注册


## 服务发现


## 负载均衡算法

- 带权重的roundrobin算法 

- random算法

- ketama算法

- P2C算法


## 服务访问



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

## discuss

- ringbuffer@126.com
