import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_transport_v1_transport_pb from '../../../../smartcore/bos/transport/v1/transport_pb'; // proto import: "smartcore/bos/transport/v1/transport.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"


export class TransportRecord extends jspb.Message {
  getTransport(): smartcore_bos_transport_v1_transport_pb.Transport | undefined;
  setTransport(value?: smartcore_bos_transport_v1_transport_pb.Transport): TransportRecord;
  hasTransport(): boolean;
  clearTransport(): TransportRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): TransportRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): TransportRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TransportRecord.AsObject;
  static toObject(includeInstance: boolean, msg: TransportRecord): TransportRecord.AsObject;
  static serializeBinaryToWriter(message: TransportRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TransportRecord;
  static deserializeBinaryFromReader(message: TransportRecord, reader: jspb.BinaryReader): TransportRecord;
}

export namespace TransportRecord {
  export type AsObject = {
    transport?: smartcore_bos_transport_v1_transport_pb.Transport.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ListTransportHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListTransportHistoryRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListTransportHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListTransportHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListTransportHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListTransportHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListTransportHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListTransportHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListTransportHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListTransportHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListTransportHistoryRequest): ListTransportHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListTransportHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListTransportHistoryRequest;
  static deserializeBinaryFromReader(message: ListTransportHistoryRequest, reader: jspb.BinaryReader): ListTransportHistoryRequest;
}

export namespace ListTransportHistoryRequest {
  export type AsObject = {
    name: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListTransportHistoryResponse extends jspb.Message {
  getTransportRecordsList(): Array<TransportRecord>;
  setTransportRecordsList(value: Array<TransportRecord>): ListTransportHistoryResponse;
  clearTransportRecordsList(): ListTransportHistoryResponse;
  addTransportRecords(value?: TransportRecord, index?: number): TransportRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListTransportHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListTransportHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListTransportHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListTransportHistoryResponse): ListTransportHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListTransportHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListTransportHistoryResponse;
  static deserializeBinaryFromReader(message: ListTransportHistoryResponse, reader: jspb.BinaryReader): ListTransportHistoryResponse;
}

export namespace ListTransportHistoryResponse {
  export type AsObject = {
    transportRecordsList: Array<TransportRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

