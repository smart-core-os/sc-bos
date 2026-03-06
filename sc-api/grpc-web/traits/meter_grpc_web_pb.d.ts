import * as grpcWeb from 'grpc-web';

import * as traits_meter_pb from '../traits/meter_pb'; // proto import: "traits/meter.proto"


export class MeterApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getMeterReading(
    request: traits_meter_pb.GetMeterReadingRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_meter_pb.MeterReading) => void
  ): grpcWeb.ClientReadableStream<traits_meter_pb.MeterReading>;

  pullMeterReadings(
    request: traits_meter_pb.PullMeterReadingsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_meter_pb.PullMeterReadingsResponse>;

}

export class MeterInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeMeterReading(
    request: traits_meter_pb.DescribeMeterReadingRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_meter_pb.MeterReadingSupport) => void
  ): grpcWeb.ClientReadableStream<traits_meter_pb.MeterReadingSupport>;

}

export class MeterApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getMeterReading(
    request: traits_meter_pb.GetMeterReadingRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_meter_pb.MeterReading>;

  pullMeterReadings(
    request: traits_meter_pb.PullMeterReadingsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_meter_pb.PullMeterReadingsResponse>;

}

export class MeterInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeMeterReading(
    request: traits_meter_pb.DescribeMeterReadingRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_meter_pb.MeterReadingSupport>;

}

