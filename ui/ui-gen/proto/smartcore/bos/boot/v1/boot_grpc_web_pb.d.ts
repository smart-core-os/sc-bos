import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_boot_v1_boot_pb from '../../../../smartcore/bos/boot/v1/boot_pb'; // proto import: "smartcore/bos/boot/v1/boot.proto"


export class BootApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getBootState(
    request: smartcore_bos_boot_v1_boot_pb.GetBootStateRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_boot_v1_boot_pb.BootState) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_boot_v1_boot_pb.BootState>;

  pullBootState(
    request: smartcore_bos_boot_v1_boot_pb.PullBootStateRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_boot_v1_boot_pb.PullBootStateResponse>;

  reboot(
    request: smartcore_bos_boot_v1_boot_pb.RebootRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_boot_v1_boot_pb.RebootResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_boot_v1_boot_pb.RebootResponse>;

}

export class BootApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getBootState(
    request: smartcore_bos_boot_v1_boot_pb.GetBootStateRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_boot_v1_boot_pb.BootState>;

  pullBootState(
    request: smartcore_bos_boot_v1_boot_pb.PullBootStateRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_boot_v1_boot_pb.PullBootStateResponse>;

  reboot(
    request: smartcore_bos_boot_v1_boot_pb.RebootRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_boot_v1_boot_pb.RebootResponse>;

}

