package enums

type CommonType string

const (
	SERVER_ZLOG_NAME CommonType = "consul-server"

	CONSUL_HealthCheckType_RPC  = "grpc"
	CONSUL_HealthCheckType_HTTP = "http"
	CONSUL_HealthCheckType_TTL  = "ttl"
)
