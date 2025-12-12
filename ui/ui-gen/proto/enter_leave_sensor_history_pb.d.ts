import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"
import * as traits_enter_leave_sensor_pb from '@smart-core-os/sc-api-grpc-web/traits/enter_leave_sensor_pb'; // proto import: "traits/enter_leave_sensor.proto"


export class EnterLeaveEventRecord extends jspb.Message {
  getEnterLeaveEvent(): traits_enter_leave_sensor_pb.EnterLeaveEvent | undefined;
  setEnterLeaveEvent(value?: traits_enter_leave_sensor_pb.EnterLeaveEvent): EnterLeaveEventRecord;
  hasEnterLeaveEvent(): boolean;
  clearEnterLeaveEvent(): EnterLeaveEventRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): EnterLeaveEventRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): EnterLeaveEventRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): EnterLeaveEventRecord.AsObject;
  static toObject(includeInstance: boolean, msg: EnterLeaveEventRecord): EnterLeaveEventRecord.AsObject;
  static serializeBinaryToWriter(message: EnterLeaveEventRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): EnterLeaveEventRecord;
  static deserializeBinaryFromReader(message: EnterLeaveEventRecord, reader: jspb.BinaryReader): EnterLeaveEventRecord;
}

export namespace EnterLeaveEventRecord {
  export type AsObject = {
    enterLeaveEvent?: traits_enter_leave_sensor_pb.EnterLeaveEvent.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ListEnterLeaveHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListEnterLeaveHistoryRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListEnterLeaveHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListEnterLeaveHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListEnterLeaveHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListEnterLeaveHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListEnterLeaveHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListEnterLeaveHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListEnterLeaveHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListEnterLeaveHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListEnterLeaveHistoryRequest): ListEnterLeaveHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListEnterLeaveHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListEnterLeaveHistoryRequest;
  static deserializeBinaryFromReader(message: ListEnterLeaveHistoryRequest, reader: jspb.BinaryReader): ListEnterLeaveHistoryRequest;
}

export namespace ListEnterLeaveHistoryRequest {
  export type AsObject = {
    name: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListEnterLeaveHistoryResponse extends jspb.Message {
  getEnterLeaveRecordsList(): Array<EnterLeaveEventRecord>;
  setEnterLeaveRecordsList(value: Array<EnterLeaveEventRecord>): ListEnterLeaveHistoryResponse;
  clearEnterLeaveRecordsList(): ListEnterLeaveHistoryResponse;
  addEnterLeaveRecords(value?: EnterLeaveEventRecord, index?: number): EnterLeaveEventRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListEnterLeaveHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListEnterLeaveHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListEnterLeaveHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListEnterLeaveHistoryResponse): ListEnterLeaveHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListEnterLeaveHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListEnterLeaveHistoryResponse;
  static deserializeBinaryFromReader(message: ListEnterLeaveHistoryResponse, reader: jspb.BinaryReader): ListEnterLeaveHistoryResponse;
}

export namespace ListEnterLeaveHistoryResponse {
  export type AsObject = {
    enterLeaveRecordsList: Array<EnterLeaveEventRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

