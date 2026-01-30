import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_meter_v1_meter_history_pb from '../../../../smartcore/bos/meter/v1/meter_history_pb'; // proto import: "smartcore/bos/meter/v1/meter_history.proto"


export class MeterHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listMeterReadingHistory(
    request: smartcore_bos_meter_v1_meter_history_pb.ListMeterReadingHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_meter_v1_meter_history_pb.ListMeterReadingHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_meter_v1_meter_history_pb.ListMeterReadingHistoryResponse>;

}

export class MeterHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listMeterReadingHistory(
    request: smartcore_bos_meter_v1_meter_history_pb.ListMeterReadingHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_meter_v1_meter_history_pb.ListMeterReadingHistoryResponse>;

}

