import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_status_v1_status_pb from '../../../../smartcore/bos/status/v1/status_pb'; // proto import: "smartcore/bos/status/v1/status.proto"


export class StatusApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getCurrentStatus(
    request: smartcore_bos_status_v1_status_pb.GetCurrentStatusRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_status_v1_status_pb.StatusLog) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_status_v1_status_pb.StatusLog>;

  pullCurrentStatus(
    request: smartcore_bos_status_v1_status_pb.PullCurrentStatusRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_status_v1_status_pb.PullCurrentStatusResponse>;

}

export class StatusApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getCurrentStatus(
    request: smartcore_bos_status_v1_status_pb.GetCurrentStatusRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_status_v1_status_pb.StatusLog>;

  pullCurrentStatus(
    request: smartcore_bos_status_v1_status_pb.PullCurrentStatusRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_status_v1_status_pb.PullCurrentStatusResponse>;

}

