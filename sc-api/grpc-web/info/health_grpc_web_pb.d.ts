import * as grpcWeb from 'grpc-web';

import * as info_health_pb from '../info/health_pb'; // proto import: "info/health.proto"


export class HealthClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getHealthState(
    request: info_health_pb.GetHealthStateRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: info_health_pb.HealthState) => void
  ): grpcWeb.ClientReadableStream<info_health_pb.HealthState>;

  pullHealthStates(
    request: info_health_pb.PullHealthStatesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<info_health_pb.PullHealthStatesResponse>;

}

export class HealthPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getHealthState(
    request: info_health_pb.GetHealthStateRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<info_health_pb.HealthState>;

  pullHealthStates(
    request: info_health_pb.PullHealthStatesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<info_health_pb.PullHealthStatesResponse>;

}

