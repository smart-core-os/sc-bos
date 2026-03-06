import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_resourceutilisation_v1_resource_utilisation_history_pb from '../../../../smartcore/bos/resourceutilisation/v1/resource_utilisation_history_pb'; // proto import: "smartcore/bos/resourceutilisation/v1/resource_utilisation_history.proto"


export class ResourceUtilisationHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listResourceUtilisationHistory(
    request: smartcore_bos_resourceutilisation_v1_resource_utilisation_history_pb.ListResourceUtilisationHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_resourceutilisation_v1_resource_utilisation_history_pb.ListResourceUtilisationHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_resourceutilisation_v1_resource_utilisation_history_pb.ListResourceUtilisationHistoryResponse>;

}

export class ResourceUtilisationHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listResourceUtilisationHistory(
    request: smartcore_bos_resourceutilisation_v1_resource_utilisation_history_pb.ListResourceUtilisationHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_resourceutilisation_v1_resource_utilisation_history_pb.ListResourceUtilisationHistoryResponse>;

}

