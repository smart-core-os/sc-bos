// Package supervisor connects BOS to the Supervisor's gRPC API, used to manage software updates. The
// Supervisor is a separate service that installs updates out-of-process and can roll them back
// automatically if the new version is unhealthy.
//
// Callers talk to the Supervisor through the generated supervisorpb.SupervisorApiClient built on the
// connection from Dial. RunStartupCommit orchestrates the once-per-boot Commit.
package supervisor

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Dial opens a lazy gRPC connection to the Supervisor's Unix socket at socketPath. grpc.NewClient is
// lazy, so the socket need not exist at the time of this call. Callers build a
// supervisorpb.SupervisorApiClient on the returned connection and Close it when done; each RPC is
// bounded by the deadline on the context the caller passes.
func Dial(socketPath string) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		"unix:"+socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}
