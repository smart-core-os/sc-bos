import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_info_v1_auth_pb from '../../../../smartcore/bos/info/v1/auth_pb'; // proto import: "smartcore/bos/info/v1/auth.proto"


export class AuthProviderClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  addAccount(
    request: smartcore_bos_info_v1_auth_pb.AddAccountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_info_v1_auth_pb.AddAccountResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_info_v1_auth_pb.AddAccountResponse>;

  removeAccount(
    request: smartcore_bos_info_v1_auth_pb.RemoveAccountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_info_v1_auth_pb.RemoveAccountResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_info_v1_auth_pb.RemoveAccountResponse>;

  updateAccountPermissions(
    request: smartcore_bos_info_v1_auth_pb.UpdateAccountPermissionsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_info_v1_auth_pb.UpdateAccountPermissionsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_info_v1_auth_pb.UpdateAccountPermissionsResponse>;

  generateToken(
    request: smartcore_bos_info_v1_auth_pb.GenerateTokenRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_info_v1_auth_pb.GenerateTokenResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_info_v1_auth_pb.GenerateTokenResponse>;

}

export class AuthProviderPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  addAccount(
    request: smartcore_bos_info_v1_auth_pb.AddAccountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_info_v1_auth_pb.AddAccountResponse>;

  removeAccount(
    request: smartcore_bos_info_v1_auth_pb.RemoveAccountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_info_v1_auth_pb.RemoveAccountResponse>;

  updateAccountPermissions(
    request: smartcore_bos_info_v1_auth_pb.UpdateAccountPermissionsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_info_v1_auth_pb.UpdateAccountPermissionsResponse>;

  generateToken(
    request: smartcore_bos_info_v1_auth_pb.GenerateTokenRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_info_v1_auth_pb.GenerateTokenResponse>;

}

