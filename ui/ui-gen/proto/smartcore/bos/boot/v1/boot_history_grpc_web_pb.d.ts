import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_boot_v1_boot_history_pb from '../../../../smartcore/bos/boot/v1/boot_history_pb'; // proto import: "smartcore/bos/boot/v1/boot_history.proto"


export class BootHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listBootEvents(
    request: smartcore_bos_boot_v1_boot_history_pb.ListBootEventsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_boot_v1_boot_history_pb.ListBootEventsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_boot_v1_boot_history_pb.ListBootEventsResponse>;

}

export class BootHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listBootEvents(
    request: smartcore_bos_boot_v1_boot_history_pb.ListBootEventsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_boot_v1_boot_history_pb.ListBootEventsResponse>;

}

