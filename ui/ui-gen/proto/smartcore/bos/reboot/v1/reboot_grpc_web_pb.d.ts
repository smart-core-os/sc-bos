import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_reboot_v1_reboot_pb from '../../../../smartcore/bos/reboot/v1/reboot_pb'; // proto import: "smartcore/bos/reboot/v1/reboot.proto"


export class RebootApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getRebootState(
    request: smartcore_bos_reboot_v1_reboot_pb.GetRebootStateRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_reboot_v1_reboot_pb.RebootState) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_reboot_v1_reboot_pb.RebootState>;

  pullRebootState(
    request: smartcore_bos_reboot_v1_reboot_pb.PullRebootStateRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_reboot_v1_reboot_pb.PullRebootStateResponse>;

  reboot(
    request: smartcore_bos_reboot_v1_reboot_pb.RebootRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_reboot_v1_reboot_pb.RebootResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_reboot_v1_reboot_pb.RebootResponse>;

}

export class RebootApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getRebootState(
    request: smartcore_bos_reboot_v1_reboot_pb.GetRebootStateRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_reboot_v1_reboot_pb.RebootState>;

  pullRebootState(
    request: smartcore_bos_reboot_v1_reboot_pb.PullRebootStateRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_reboot_v1_reboot_pb.PullRebootStateResponse>;

  reboot(
    request: smartcore_bos_reboot_v1_reboot_pb.RebootRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_reboot_v1_reboot_pb.RebootResponse>;

}

