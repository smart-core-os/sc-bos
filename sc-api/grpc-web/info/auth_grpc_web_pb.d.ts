import * as grpcWeb from 'grpc-web';

import * as info_auth_pb from '../info/auth_pb'; // proto import: "info/auth.proto"


export class AuthProviderClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  addAccount(
    request: info_auth_pb.AddAccountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: info_auth_pb.AddAccountResponse) => void
  ): grpcWeb.ClientReadableStream<info_auth_pb.AddAccountResponse>;

  removeAccount(
    request: info_auth_pb.RemoveAccountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: info_auth_pb.RemoveAccountResponse) => void
  ): grpcWeb.ClientReadableStream<info_auth_pb.RemoveAccountResponse>;

  updateAccountPermissions(
    request: info_auth_pb.UpdateAccountPermissionsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: info_auth_pb.UpdateAccountPermissionsResponse) => void
  ): grpcWeb.ClientReadableStream<info_auth_pb.UpdateAccountPermissionsResponse>;

  generateToken(
    request: info_auth_pb.GenerateTokenRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: info_auth_pb.GenerateTokenResponse) => void
  ): grpcWeb.ClientReadableStream<info_auth_pb.GenerateTokenResponse>;

}

export class AuthProviderPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  addAccount(
    request: info_auth_pb.AddAccountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<info_auth_pb.AddAccountResponse>;

  removeAccount(
    request: info_auth_pb.RemoveAccountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<info_auth_pb.RemoveAccountResponse>;

  updateAccountPermissions(
    request: info_auth_pb.UpdateAccountPermissionsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<info_auth_pb.UpdateAccountPermissionsResponse>;

  generateToken(
    request: info_auth_pb.GenerateTokenRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<info_auth_pb.GenerateTokenResponse>;

}

