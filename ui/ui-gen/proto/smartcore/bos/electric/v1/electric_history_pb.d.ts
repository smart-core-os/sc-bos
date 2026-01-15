import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as traits_electric_pb from '@smart-core-os/sc-api-grpc-web/traits/electric_pb'; // proto import: "traits/electric.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"


export class ElectricDemandRecord extends jspb.Message {
  getElectricDemand(): traits_electric_pb.ElectricDemand | undefined;
  setElectricDemand(value?: traits_electric_pb.ElectricDemand): ElectricDemandRecord;
  hasElectricDemand(): boolean;
  clearElectricDemand(): ElectricDemandRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): ElectricDemandRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): ElectricDemandRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ElectricDemandRecord.AsObject;
  static toObject(includeInstance: boolean, msg: ElectricDemandRecord): ElectricDemandRecord.AsObject;
  static serializeBinaryToWriter(message: ElectricDemandRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ElectricDemandRecord;
  static deserializeBinaryFromReader(message: ElectricDemandRecord, reader: jspb.BinaryReader): ElectricDemandRecord;
}

export namespace ElectricDemandRecord {
  export type AsObject = {
    electricDemand?: traits_electric_pb.ElectricDemand.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ListElectricDemandHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListElectricDemandHistoryRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListElectricDemandHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListElectricDemandHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListElectricDemandHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListElectricDemandHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListElectricDemandHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListElectricDemandHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListElectricDemandHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListElectricDemandHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListElectricDemandHistoryRequest): ListElectricDemandHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListElectricDemandHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListElectricDemandHistoryRequest;
  static deserializeBinaryFromReader(message: ListElectricDemandHistoryRequest, reader: jspb.BinaryReader): ListElectricDemandHistoryRequest;
}

export namespace ListElectricDemandHistoryRequest {
  export type AsObject = {
    name: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListElectricDemandHistoryResponse extends jspb.Message {
  getElectricDemandRecordsList(): Array<ElectricDemandRecord>;
  setElectricDemandRecordsList(value: Array<ElectricDemandRecord>): ListElectricDemandHistoryResponse;
  clearElectricDemandRecordsList(): ListElectricDemandHistoryResponse;
  addElectricDemandRecords(value?: ElectricDemandRecord, index?: number): ElectricDemandRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListElectricDemandHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListElectricDemandHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListElectricDemandHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListElectricDemandHistoryResponse): ListElectricDemandHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListElectricDemandHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListElectricDemandHistoryResponse;
  static deserializeBinaryFromReader(message: ListElectricDemandHistoryResponse, reader: jspb.BinaryReader): ListElectricDemandHistoryResponse;
}

export namespace ListElectricDemandHistoryResponse {
  export type AsObject = {
    electricDemandRecordsList: Array<ElectricDemandRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

