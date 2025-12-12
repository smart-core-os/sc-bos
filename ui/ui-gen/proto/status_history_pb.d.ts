import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"
import * as status_pb from './status_pb'; // proto import: "status.proto"


export class StatusLogRecord extends jspb.Message {
  getCurrentStatus(): status_pb.StatusLog | undefined;
  setCurrentStatus(value?: status_pb.StatusLog): StatusLogRecord;
  hasCurrentStatus(): boolean;
  clearCurrentStatus(): StatusLogRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): StatusLogRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): StatusLogRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StatusLogRecord.AsObject;
  static toObject(includeInstance: boolean, msg: StatusLogRecord): StatusLogRecord.AsObject;
  static serializeBinaryToWriter(message: StatusLogRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StatusLogRecord;
  static deserializeBinaryFromReader(message: StatusLogRecord, reader: jspb.BinaryReader): StatusLogRecord;
}

export namespace StatusLogRecord {
  export type AsObject = {
    currentStatus?: status_pb.StatusLog.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ListCurrentStatusHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListCurrentStatusHistoryRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListCurrentStatusHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListCurrentStatusHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListCurrentStatusHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListCurrentStatusHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListCurrentStatusHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListCurrentStatusHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListCurrentStatusHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListCurrentStatusHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListCurrentStatusHistoryRequest): ListCurrentStatusHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListCurrentStatusHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListCurrentStatusHistoryRequest;
  static deserializeBinaryFromReader(message: ListCurrentStatusHistoryRequest, reader: jspb.BinaryReader): ListCurrentStatusHistoryRequest;
}

export namespace ListCurrentStatusHistoryRequest {
  export type AsObject = {
    name: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListCurrentStatusHistoryResponse extends jspb.Message {
  getCurrentStatusRecordsList(): Array<StatusLogRecord>;
  setCurrentStatusRecordsList(value: Array<StatusLogRecord>): ListCurrentStatusHistoryResponse;
  clearCurrentStatusRecordsList(): ListCurrentStatusHistoryResponse;
  addCurrentStatusRecords(value?: StatusLogRecord, index?: number): StatusLogRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListCurrentStatusHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListCurrentStatusHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListCurrentStatusHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListCurrentStatusHistoryResponse): ListCurrentStatusHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListCurrentStatusHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListCurrentStatusHistoryResponse;
  static deserializeBinaryFromReader(message: ListCurrentStatusHistoryResponse, reader: jspb.BinaryReader): ListCurrentStatusHistoryResponse;
}

export namespace ListCurrentStatusHistoryResponse {
  export type AsObject = {
    currentStatusRecordsList: Array<StatusLogRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

