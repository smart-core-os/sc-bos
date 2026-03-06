import * as grpcWeb from 'grpc-web';

import * as traits_access_pb from '../traits/access_pb'; // proto import: "traits/access.proto"


export class AccessApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getLastAccessAttempt(
    request: traits_access_pb.GetLastAccessAttemptRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_access_pb.AccessAttempt) => void
  ): grpcWeb.ClientReadableStream<traits_access_pb.AccessAttempt>;

  pullAccessAttempts(
    request: traits_access_pb.PullAccessAttemptsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_access_pb.PullAccessAttemptsResponse>;

}

export class AccessApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getLastAccessAttempt(
    request: traits_access_pb.GetLastAccessAttemptRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_access_pb.AccessAttempt>;

  pullAccessAttempts(
    request: traits_access_pb.PullAccessAttemptsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_access_pb.PullAccessAttemptsResponse>;

}

