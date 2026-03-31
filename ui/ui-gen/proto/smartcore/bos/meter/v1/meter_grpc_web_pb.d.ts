import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_meter_v1_meter_pb from '../../../../smartcore/bos/meter/v1/meter_pb'; // proto import: "smartcore/bos/meter/v1/meter.proto"


export class MeterApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getMeterReading(
    request: smartcore_bos_meter_v1_meter_pb.GetMeterReadingRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_meter_v1_meter_pb.MeterReading) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_meter_v1_meter_pb.MeterReading>;

  pullMeterReadings(
    request: smartcore_bos_meter_v1_meter_pb.PullMeterReadingsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_meter_v1_meter_pb.PullMeterReadingsResponse>;

}

export class MeterInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeMeterReading(
    request: smartcore_bos_meter_v1_meter_pb.DescribeMeterReadingRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_meter_v1_meter_pb.MeterReadingSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_meter_v1_meter_pb.MeterReadingSupport>;

}

export class MeterApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getMeterReading(
    request: smartcore_bos_meter_v1_meter_pb.GetMeterReadingRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_meter_v1_meter_pb.MeterReading>;

  pullMeterReadings(
    request: smartcore_bos_meter_v1_meter_pb.PullMeterReadingsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_meter_v1_meter_pb.PullMeterReadingsResponse>;

}

export class MeterInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeMeterReading(
    request: smartcore_bos_meter_v1_meter_pb.DescribeMeterReadingRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_meter_v1_meter_pb.MeterReadingSupport>;

}

