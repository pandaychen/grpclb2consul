package healthcheck

//实现grpc服务的健康检查

import (
	"context"
	"fmt"
	"log"

	pb "google.golang.org/grpc/health/grpc_health_v1"
)

type HealthyCheck struct{}

func New() *HealthyCheck {
	return &HealthyCheck{}
}

/*
	type HealthClient interface {
    // If the requested service is unknown, the call will fail with status
    // NOT_FOUND.
    Check(ctx context.Context, in *HealthCheckRequest, opts ...grpc.CallOption) (*HealthCheckResponse, error)
    // Performs a watch for the serving status of the requested service.
    // The server will immediately send back a message indicating the current
    // serving status.  It will then subsequently send a new message whenever
    // the service's serving status changes.
    //
    // If the requested service is unknown when the call is received, the
    // server will send a message setting the serving status to
    // SERVICE_UNKNOWN but will *not* terminate the call.  If at some
    // future point, the serving status of the service becomes known, the
    // server will send a new message with the service's serving status.
    //
    // If the call terminates with status UNIMPLEMENTED, then clients
    // should assume this method is not supported and should not retry the
    // call.  If the call terminates with any other status (including OK),
    // clients should retry the call with appropriate exponential backoff.
    Watch(ctx context.Context, in *HealthCheckRequest, opts ...grpc.CallOption) (Health_WatchClient, error)
}
HealthClient is the client API for Health service.
*/

func (h *HealthyCheck) Check(ctx context.Context, in *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	/*
		log.Printf("service check:%s", in.Service)
		var s pb.HealthCheckResponse_ServingStatus = 1
		return &pb.HealthCheckResponse{
			Status: s,
		}, nil
	*/
	//more check method/logic could be add
	fmt.Println("call health check", in.Service)
	return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}, nil
	//return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_NOT_SERVING }, nil
}
func (h *HealthyCheck) Watch(in *pb.HealthCheckRequest, w pb.Health_WatchServer) error {
	log.Printf("service watch:%s", in.Service)
	var s pb.HealthCheckResponse_ServingStatus = 1
	r := &pb.HealthCheckResponse{
		Status: s,
	}
	for {
		w.Send(r)
	}
	return nil
}
