import * as grpcWeb from 'grpc-web';

import * as info_info_pb from '../info/info_pb'; // proto import: "info/info.proto"


export class InfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listDevices(
    request: info_info_pb.ListDevicesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: info_info_pb.ListDevicesResponse) => void
  ): grpcWeb.ClientReadableStream<info_info_pb.ListDevicesResponse>;

  pullDevices(
    request: info_info_pb.PullDevicesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<info_info_pb.PullDevicesResponse>;

}

export class InfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listDevices(
    request: info_info_pb.ListDevicesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<info_info_pb.ListDevicesResponse>;

  pullDevices(
    request: info_info_pb.PullDevicesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<info_info_pb.PullDevicesResponse>;

}

