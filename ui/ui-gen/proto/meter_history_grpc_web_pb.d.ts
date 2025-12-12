import * as grpcWeb from 'grpc-web';

import * as meter_history_pb from './meter_history_pb'; // proto import: "meter_history.proto"


export class MeterHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listMeterReadingHistory(
    request: meter_history_pb.ListMeterReadingHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: meter_history_pb.ListMeterReadingHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<meter_history_pb.ListMeterReadingHistoryResponse>;

}

export class MeterHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listMeterReadingHistory(
    request: meter_history_pb.ListMeterReadingHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<meter_history_pb.ListMeterReadingHistoryResponse>;

}

