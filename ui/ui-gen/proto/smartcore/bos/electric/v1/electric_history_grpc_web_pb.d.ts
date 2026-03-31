import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_electric_v1_electric_history_pb from '../../../../smartcore/bos/electric/v1/electric_history_pb'; // proto import: "smartcore/bos/electric/v1/electric_history.proto"


export class ElectricHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listElectricDemandHistory(
    request: smartcore_bos_electric_v1_electric_history_pb.ListElectricDemandHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_electric_v1_electric_history_pb.ListElectricDemandHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_history_pb.ListElectricDemandHistoryResponse>;

}

export class ElectricHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listElectricDemandHistory(
    request: smartcore_bos_electric_v1_electric_history_pb.ListElectricDemandHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_electric_v1_electric_history_pb.ListElectricDemandHistoryResponse>;

}

