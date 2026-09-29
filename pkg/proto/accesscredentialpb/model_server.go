package accesscredentialpb

import (
	"context"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/pkg/util/masks"
)

// ModelServer adapts a Model to AccessCredentialApiServer and AccessCredentialInfoServer.
type ModelServer struct {
	UnimplementedAccessCredentialApiServer
	UnimplementedAccessCredentialInfoServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterAccessCredentialApiServer(server, m)
	RegisterAccessCredentialInfoServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) GetCredential(_ context.Context, request *GetCredentialRequest) (*Credential, error) {
	c, err := m.model.GetCredential(request.GetId())
	if err != nil {
		return nil, err
	}
	masks.NewResponseFilter(masks.WithFieldMask(request.GetReadMask())).Filter(c)
	return c, nil
}

func (m *ModelServer) ListCredentials(_ context.Context, request *ListCredentialsRequest) (*ListCredentialsResponse, error) {
	return ListPage(m.model.ListCredentials(), request)
}

func (m *ModelServer) CreateCredential(_ context.Context, request *CreateCredentialRequest) (*Credential, error) {
	return m.model.CreateCredential(request.GetCredential())
}

func (m *ModelServer) UpdateCredential(_ context.Context, request *UpdateCredentialRequest) (*Credential, error) {
	return m.model.UpdateCredential(request.GetCredential(), request.GetUpdateMask())
}

func (m *ModelServer) DeleteCredential(_ context.Context, request *DeleteCredentialRequest) (*DeleteCredentialResponse, error) {
	if err := m.model.DeleteCredential(request.GetId(), request.GetAllowMissing()); err != nil {
		return nil, err
	}
	return &DeleteCredentialResponse{}, nil
}

func (m *ModelServer) DescribeCredential(_ context.Context, _ *DescribeCredentialRequest) (*CredentialSupport, error) {
	return m.model.DescribeCredential(), nil
}
