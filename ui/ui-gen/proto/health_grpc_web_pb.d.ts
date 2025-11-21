import * as grpcWeb from 'grpc-web';

import * as health_pb from './health_pb'; // proto import: "health.proto"


export class HealthApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listHealthChecks(
    request: health_pb.ListHealthChecksRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: health_pb.ListHealthChecksResponse) => void
  ): grpcWeb.ClientReadableStream<health_pb.ListHealthChecksResponse>;

  pullHealthChecks(
    request: health_pb.PullHealthChecksRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<health_pb.PullHealthChecksResponse>;

  getHealthCheck(
    request: health_pb.GetHealthCheckRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: health_pb.HealthCheck) => void
  ): grpcWeb.ClientReadableStream<health_pb.HealthCheck>;

  pullHealthCheck(
    request: health_pb.PullHealthCheckRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<health_pb.PullHealthCheckResponse>;

}

export class HealthHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listHealthCheckHistory(
    request: health_pb.ListHealthCheckHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: health_pb.ListHealthCheckHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<health_pb.ListHealthCheckHistoryResponse>;

}

export class HealthApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listHealthChecks(
    request: health_pb.ListHealthChecksRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<health_pb.ListHealthChecksResponse>;

  pullHealthChecks(
    request: health_pb.PullHealthChecksRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<health_pb.PullHealthChecksResponse>;

  getHealthCheck(
    request: health_pb.GetHealthCheckRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<health_pb.HealthCheck>;

  pullHealthCheck(
    request: health_pb.PullHealthCheckRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<health_pb.PullHealthCheckResponse>;

}

export class HealthHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listHealthCheckHistory(
    request: health_pb.ListHealthCheckHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<health_pb.ListHealthCheckHistoryResponse>;

}

