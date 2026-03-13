package wrap

import (
	"context"
	"fmt"

	testproto2 "github.com/smart-core-os/sc-bos/internal/testproto"
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
	conn := ServerToClient(testproto2.TestApi_ServiceDesc, srv)
	client := testproto2.NewTestApiClient(conn)

	res, err := client.Unary(context.Background(), &testproto2.UnaryRequest{Msg: "hello"})
	if err != nil {
		panic(err)
	}
	fmt.Println(res.Msg)
	// Output: hello
}

type exampleServer struct {
	testproto2.UnimplementedTestApiServer
}

func (s *exampleServer) Unary(ctx context.Context, req *testproto2.UnaryRequest) (*testproto2.UnaryResponse, error) {
	return &testproto2.UnaryResponse{Msg: req.Msg}, nil
}
