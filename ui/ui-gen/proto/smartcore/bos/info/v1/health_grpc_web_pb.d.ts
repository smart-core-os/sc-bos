import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_info_v1_health_pb from '../../../../smartcore/bos/info/v1/health_pb'; // proto import: "smartcore/bos/info/v1/health.proto"


export class HealthClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getHealthState(
    request: smartcore_bos_info_v1_health_pb.GetHealthStateRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_info_v1_health_pb.HealthState) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_info_v1_health_pb.HealthState>;

  pullHealthStates(
    request: smartcore_bos_info_v1_health_pb.PullHealthStatesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_info_v1_health_pb.PullHealthStatesResponse>;

}

export class HealthPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getHealthState(
    request: smartcore_bos_info_v1_health_pb.GetHealthStateRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_info_v1_health_pb.HealthState>;

  pullHealthStates(
    request: smartcore_bos_info_v1_health_pb.PullHealthStatesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_info_v1_health_pb.PullHealthStatesResponse>;

}

