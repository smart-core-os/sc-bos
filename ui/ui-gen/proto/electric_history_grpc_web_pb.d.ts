import * as grpcWeb from 'grpc-web';

import * as electric_history_pb from './electric_history_pb'; // proto import: "electric_history.proto"


export class ElectricHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listElectricDemandHistory(
    request: electric_history_pb.ListElectricDemandHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: electric_history_pb.ListElectricDemandHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<electric_history_pb.ListElectricDemandHistoryResponse>;

}

export class ElectricHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listElectricDemandHistory(
    request: electric_history_pb.ListElectricDemandHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<electric_history_pb.ListElectricDemandHistoryResponse>;

}

