package wrap

import (
	"context"
	"fmt"

	"github.com/smart-core-os/sc-golang/internal/testproto"
)

// Given the following gRPC service definition:
//
//     syntax = "proto3";
//     service TestApi {
//     	 rpc Unary(UnaryRequest) returns (UnaryResponse);
//     }
//     message UnaryRequest {
//     	 string msg = 1;
//     }
//     message UnaryResponse {
//     	 string msg = 1;
//     }

func ExampleServerToClient() {
	srv := &exampleServer{}
	conn := ServerToClient(testproto.TestApi_ServiceDesc, srv)
	client := testproto.NewTestApiClient(conn)

	res, err := client.Unary(context.Background(), &testproto.UnaryRequest{Msg: "hello"})
	if err != nil {
		panic(err)
	}
	fmt.Println(res.Msg)
	// Output: hello
}

type exampleServer struct {
	testproto.UnimplementedTestApiServer
}

func (s *exampleServer) Unary(ctx context.Context, req *testproto.UnaryRequest) (*testproto.UnaryResponse, error) {
	return &testproto.UnaryResponse{Msg: req.Msg}, nil
}
