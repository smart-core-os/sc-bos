import * as grpcWeb from 'grpc-web';

import * as access_pb from './access_pb'; // proto import: "access.proto"


export class AccessApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getLastAccessAttempt(
    request: access_pb.GetLastAccessAttemptRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: access_pb.AccessAttempt) => void
  ): grpcWeb.ClientReadableStream<access_pb.AccessAttempt>;

  pullAccessAttempts(
    request: access_pb.PullAccessAttemptsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<access_pb.PullAccessAttemptsResponse>;

  createAccessGrant(
    request: access_pb.CreateAccessGrantRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: access_pb.AccessGrant) => void
  ): grpcWeb.ClientReadableStream<access_pb.AccessGrant>;

  updateAccessGrant(
    request: access_pb.UpdateAccessGrantRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: access_pb.AccessGrant) => void
  ): grpcWeb.ClientReadableStream<access_pb.AccessGrant>;

  deleteAccessGrant(
    request: access_pb.DeleteAccessGrantRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: access_pb.DeleteAccessGrantResponse) => void
  ): grpcWeb.ClientReadableStream<access_pb.DeleteAccessGrantResponse>;

  getAccessGrant(
    request: access_pb.GetAccessGrantsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: access_pb.AccessGrant) => void
  ): grpcWeb.ClientReadableStream<access_pb.AccessGrant>;

  listAccessGrants(
    request: access_pb.ListAccessGrantsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: access_pb.ListAccessGrantsResponse) => void
  ): grpcWeb.ClientReadableStream<access_pb.ListAccessGrantsResponse>;

  createActor(
    request: access_pb.CreateActorRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: access_pb.CreateActorResponse) => void
  ): grpcWeb.ClientReadableStream<access_pb.CreateActorResponse>;

  grantAccess(
    request: access_pb.GrantAccessRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: access_pb.GrantAccessResponse) => void
  ): grpcWeb.ClientReadableStream<access_pb.GrantAccessResponse>;

  saveQRCredential(
    request: access_pb.SaveQRCredentialRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: access_pb.SaveQRCredentialResponse) => void
  ): grpcWeb.ClientReadableStream<access_pb.SaveQRCredentialResponse>;

}

export class AccessApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getLastAccessAttempt(
    request: access_pb.GetLastAccessAttemptRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<access_pb.AccessAttempt>;

  pullAccessAttempts(
    request: access_pb.PullAccessAttemptsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<access_pb.PullAccessAttemptsResponse>;

  createAccessGrant(
    request: access_pb.CreateAccessGrantRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<access_pb.AccessGrant>;

  updateAccessGrant(
    request: access_pb.UpdateAccessGrantRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<access_pb.AccessGrant>;

  deleteAccessGrant(
    request: access_pb.DeleteAccessGrantRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<access_pb.DeleteAccessGrantResponse>;

  getAccessGrant(
    request: access_pb.GetAccessGrantsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<access_pb.AccessGrant>;

  listAccessGrants(
    request: access_pb.ListAccessGrantsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<access_pb.ListAccessGrantsResponse>;

  createActor(
    request: access_pb.CreateActorRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<access_pb.CreateActorResponse>;

  grantAccess(
    request: access_pb.GrantAccessRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<access_pb.GrantAccessResponse>;

  saveQRCredential(
    request: access_pb.SaveQRCredentialRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<access_pb.SaveQRCredentialResponse>;

}

