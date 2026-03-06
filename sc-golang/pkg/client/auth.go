package client

import (
	"google.golang.org/grpc/credentials"
)

type AuthCredentials struct {
	Creds credentials.TransportCredentials
	Token string
}
