import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_health_v1_health_pb from '../../../../smartcore/bos/health/v1/health_pb'; // proto import: "smartcore/bos/health/v1/health.proto"


export class HealthApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listHealthChecks(
    request: smartcore_bos_health_v1_health_pb.ListHealthChecksRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_health_v1_health_pb.ListHealthChecksResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_health_v1_health_pb.ListHealthChecksResponse>;

  pullHealthChecks(
    request: smartcore_bos_health_v1_health_pb.PullHealthChecksRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_health_v1_health_pb.PullHealthChecksResponse>;

  getHealthCheck(
    request: smartcore_bos_health_v1_health_pb.GetHealthCheckRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_health_v1_health_pb.HealthCheck) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_health_v1_health_pb.HealthCheck>;

  pullHealthCheck(
    request: smartcore_bos_health_v1_health_pb.PullHealthCheckRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_health_v1_health_pb.PullHealthCheckResponse>;

}

export class HealthApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listHealthChecks(
    request: smartcore_bos_health_v1_health_pb.ListHealthChecksRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_health_v1_health_pb.ListHealthChecksResponse>;

  pullHealthChecks(
    request: smartcore_bos_health_v1_health_pb.PullHealthChecksRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_health_v1_health_pb.PullHealthChecksResponse>;

  getHealthCheck(
    request: smartcore_bos_health_v1_health_pb.GetHealthCheckRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_health_v1_health_pb.HealthCheck>;

  pullHealthCheck(
    request: smartcore_bos_health_v1_health_pb.PullHealthCheckRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_health_v1_health_pb.PullHealthCheckResponse>;

}

