package enums

type ServiceType string

const (
	ServiceType_RPC  ServiceType = "RPC"
	ServiceType_HTTP ServiceType = "HTTP"
)

var ServiceTypeMap = map[ServiceType]string{
	ServiceType_RPC:  "rpc-service",
	ServiceType_HTTP: "http-service",
}
