package gateway

import (
	"github.com/Zgxh/grpc-gen/proto/apis"
)

const (
	Addr           = ":10000"
	GrpcServerAddr = "localhost:8090"
)

var (
	Apis = apis.GetApis()
)
