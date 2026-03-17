import * as grpcWeb from 'grpc-web';

import * as traits_waste_pb from '../traits/waste_pb'; // proto import: "traits/waste.proto"


export class WasteApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listWasteRecords(
    request: traits_waste_pb.ListWasteRecordsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_waste_pb.ListWasteRecordsResponse) => void
  ): grpcWeb.ClientReadableStream<traits_waste_pb.ListWasteRecordsResponse>;

  pullWasteRecords(
    request: traits_waste_pb.PullWasteRecordsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_waste_pb.PullWasteRecordsResponse>;

}

export class WasteInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeWasteRecord(
    request: traits_waste_pb.DescribeWasteRecordRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_waste_pb.WasteRecordSupport) => void
  ): grpcWeb.ClientReadableStream<traits_waste_pb.WasteRecordSupport>;

}

export class WasteApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listWasteRecords(
    request: traits_waste_pb.ListWasteRecordsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_waste_pb.ListWasteRecordsResponse>;

  pullWasteRecords(
    request: traits_waste_pb.PullWasteRecordsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_waste_pb.PullWasteRecordsResponse>;

}

export class WasteInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeWasteRecord(
    request: traits_waste_pb.DescribeWasteRecordRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_waste_pb.WasteRecordSupport>;

}

