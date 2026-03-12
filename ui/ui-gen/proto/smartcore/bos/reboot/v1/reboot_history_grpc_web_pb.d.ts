import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_reboot_v1_reboot_history_pb from '../../../../smartcore/bos/reboot/v1/reboot_history_pb'; // proto import: "smartcore/bos/reboot/v1/reboot_history.proto"


export class RebootHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listRebootEvents(
    request: smartcore_bos_reboot_v1_reboot_history_pb.ListRebootEventsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_reboot_v1_reboot_history_pb.ListRebootEventsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_reboot_v1_reboot_history_pb.ListRebootEventsResponse>;

}

export class RebootHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listRebootEvents(
    request: smartcore_bos_reboot_v1_reboot_history_pb.ListRebootEventsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_reboot_v1_reboot_history_pb.ListRebootEventsResponse>;

}

