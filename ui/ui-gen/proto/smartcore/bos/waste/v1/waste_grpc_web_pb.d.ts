import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_waste_v1_waste_pb from '../../../../smartcore/bos/waste/v1/waste_pb'; // proto import: "smartcore/bos/waste/v1/waste.proto"


export class WasteApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listWasteRecords(
    request: smartcore_bos_waste_v1_waste_pb.ListWasteRecordsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_waste_v1_waste_pb.ListWasteRecordsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_waste_v1_waste_pb.ListWasteRecordsResponse>;

  pullWasteRecords(
    request: smartcore_bos_waste_v1_waste_pb.PullWasteRecordsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_waste_v1_waste_pb.PullWasteRecordsResponse>;

}

export class WasteInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeWasteRecord(
    request: smartcore_bos_waste_v1_waste_pb.DescribeWasteRecordRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_waste_v1_waste_pb.WasteRecordSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_waste_v1_waste_pb.WasteRecordSupport>;

}

export class WasteApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listWasteRecords(
    request: smartcore_bos_waste_v1_waste_pb.ListWasteRecordsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_waste_v1_waste_pb.ListWasteRecordsResponse>;

  pullWasteRecords(
    request: smartcore_bos_waste_v1_waste_pb.PullWasteRecordsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_waste_v1_waste_pb.PullWasteRecordsResponse>;

}

export class WasteInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeWasteRecord(
    request: smartcore_bos_waste_v1_waste_pb.DescribeWasteRecordRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_waste_v1_waste_pb.WasteRecordSupport>;

}

