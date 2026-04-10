import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_info_v1_info_pb from '../../../../smartcore/bos/info/v1/info_pb'; // proto import: "smartcore/bos/info/v1/info.proto"


export class InfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listDevices(
    request: smartcore_bos_info_v1_info_pb.ListDevicesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_info_v1_info_pb.ListDevicesResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_info_v1_info_pb.ListDevicesResponse>;

  pullDevices(
    request: smartcore_bos_info_v1_info_pb.PullDevicesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_info_v1_info_pb.PullDevicesResponse>;

}

export class InfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listDevices(
    request: smartcore_bos_info_v1_info_pb.ListDevicesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_info_v1_info_pb.ListDevicesResponse>;

  pullDevices(
    request: smartcore_bos_info_v1_info_pb.PullDevicesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_info_v1_info_pb.PullDevicesResponse>;

}

