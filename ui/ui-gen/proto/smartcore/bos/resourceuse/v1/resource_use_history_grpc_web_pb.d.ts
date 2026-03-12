import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_resourceuse_v1_resource_use_history_pb from '../../../../smartcore/bos/resourceuse/v1/resource_use_history_pb'; // proto import: "smartcore/bos/resourceuse/v1/resource_use_history.proto"


export class ResourceUseHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listResourceUseHistory(
    request: smartcore_bos_resourceuse_v1_resource_use_history_pb.ListResourceUseHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_resourceuse_v1_resource_use_history_pb.ListResourceUseHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_resourceuse_v1_resource_use_history_pb.ListResourceUseHistoryResponse>;

}

export class ResourceUseHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listResourceUseHistory(
    request: smartcore_bos_resourceuse_v1_resource_use_history_pb.ListResourceUseHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_resourceuse_v1_resource_use_history_pb.ListResourceUseHistoryResponse>;

}

