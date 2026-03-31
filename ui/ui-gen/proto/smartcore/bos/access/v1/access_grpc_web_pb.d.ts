import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_access_v1_access_pb from '../../../../smartcore/bos/access/v1/access_pb'; // proto import: "smartcore/bos/access/v1/access.proto"


export class AccessApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getLastAccessAttempt(
    request: smartcore_bos_access_v1_access_pb.GetLastAccessAttemptRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_access_v1_access_pb.AccessAttempt) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_access_v1_access_pb.AccessAttempt>;

  pullAccessAttempts(
    request: smartcore_bos_access_v1_access_pb.PullAccessAttemptsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_access_v1_access_pb.PullAccessAttemptsResponse>;

  createAccessGrant(
    request: smartcore_bos_access_v1_access_pb.CreateAccessGrantRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_access_v1_access_pb.AccessGrant) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_access_v1_access_pb.AccessGrant>;

  updateAccessGrant(
    request: smartcore_bos_access_v1_access_pb.UpdateAccessGrantRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_access_v1_access_pb.AccessGrant) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_access_v1_access_pb.AccessGrant>;

  deleteAccessGrant(
    request: smartcore_bos_access_v1_access_pb.DeleteAccessGrantRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_access_v1_access_pb.DeleteAccessGrantResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_access_v1_access_pb.DeleteAccessGrantResponse>;

  getAccessGrant(
    request: smartcore_bos_access_v1_access_pb.GetAccessGrantsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_access_v1_access_pb.AccessGrant) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_access_v1_access_pb.AccessGrant>;

  listAccessGrants(
    request: smartcore_bos_access_v1_access_pb.ListAccessGrantsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_access_v1_access_pb.ListAccessGrantsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_access_v1_access_pb.ListAccessGrantsResponse>;

}

export class AccessApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getLastAccessAttempt(
    request: smartcore_bos_access_v1_access_pb.GetLastAccessAttemptRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_access_v1_access_pb.AccessAttempt>;

  pullAccessAttempts(
    request: smartcore_bos_access_v1_access_pb.PullAccessAttemptsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_access_v1_access_pb.PullAccessAttemptsResponse>;

  createAccessGrant(
    request: smartcore_bos_access_v1_access_pb.CreateAccessGrantRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_access_v1_access_pb.AccessGrant>;

  updateAccessGrant(
    request: smartcore_bos_access_v1_access_pb.UpdateAccessGrantRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_access_v1_access_pb.AccessGrant>;

  deleteAccessGrant(
    request: smartcore_bos_access_v1_access_pb.DeleteAccessGrantRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_access_v1_access_pb.DeleteAccessGrantResponse>;

  getAccessGrant(
    request: smartcore_bos_access_v1_access_pb.GetAccessGrantsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_access_v1_access_pb.AccessGrant>;

  listAccessGrants(
    request: smartcore_bos_access_v1_access_pb.ListAccessGrantsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_access_v1_access_pb.ListAccessGrantsResponse>;

}

