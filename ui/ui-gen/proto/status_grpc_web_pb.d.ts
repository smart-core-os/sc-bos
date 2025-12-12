import * as grpcWeb from 'grpc-web';

import * as status_pb from './status_pb'; // proto import: "status.proto"


export class StatusApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getCurrentStatus(
    request: status_pb.GetCurrentStatusRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: status_pb.StatusLog) => void
  ): grpcWeb.ClientReadableStream<status_pb.StatusLog>;

  pullCurrentStatus(
    request: status_pb.PullCurrentStatusRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<status_pb.PullCurrentStatusResponse>;

}

export class StatusApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getCurrentStatus(
    request: status_pb.GetCurrentStatusRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<status_pb.StatusLog>;

  pullCurrentStatus(
    request: status_pb.PullCurrentStatusRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<status_pb.PullCurrentStatusResponse>;

}

