import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_devices_v1_devices_pb from '../../../../smartcore/bos/devices/v1/devices_pb'; // proto import: "smartcore/bos/devices/v1/devices.proto"


export class DevicesApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listDevices(
    request: smartcore_bos_devices_v1_devices_pb.ListDevicesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_devices_v1_devices_pb.ListDevicesResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_devices_v1_devices_pb.ListDevicesResponse>;

  pullDevices(
    request: smartcore_bos_devices_v1_devices_pb.PullDevicesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_devices_v1_devices_pb.PullDevicesResponse>;

  getDevicesMetadata(
    request: smartcore_bos_devices_v1_devices_pb.GetDevicesMetadataRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_devices_v1_devices_pb.DevicesMetadata) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_devices_v1_devices_pb.DevicesMetadata>;

  pullDevicesMetadata(
    request: smartcore_bos_devices_v1_devices_pb.PullDevicesMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_devices_v1_devices_pb.PullDevicesMetadataResponse>;

  getDownloadDevicesUrl(
    request: smartcore_bos_devices_v1_devices_pb.GetDownloadDevicesUrlRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_devices_v1_devices_pb.DownloadDevicesUrl) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_devices_v1_devices_pb.DownloadDevicesUrl>;

}

export class DevicesApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listDevices(
    request: smartcore_bos_devices_v1_devices_pb.ListDevicesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_devices_v1_devices_pb.ListDevicesResponse>;

  pullDevices(
    request: smartcore_bos_devices_v1_devices_pb.PullDevicesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_devices_v1_devices_pb.PullDevicesResponse>;

  getDevicesMetadata(
    request: smartcore_bos_devices_v1_devices_pb.GetDevicesMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_devices_v1_devices_pb.DevicesMetadata>;

  pullDevicesMetadata(
    request: smartcore_bos_devices_v1_devices_pb.PullDevicesMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_devices_v1_devices_pb.PullDevicesMetadataResponse>;

  getDownloadDevicesUrl(
    request: smartcore_bos_devices_v1_devices_pb.GetDownloadDevicesUrlRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_devices_v1_devices_pb.DownloadDevicesUrl>;

}

