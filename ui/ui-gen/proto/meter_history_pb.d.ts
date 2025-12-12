import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"
import * as meter_pb from './meter_pb'; // proto import: "meter.proto"


export class MeterReadingRecord extends jspb.Message {
  getMeterReading(): meter_pb.MeterReading | undefined;
  setMeterReading(value?: meter_pb.MeterReading): MeterReadingRecord;
  hasMeterReading(): boolean;
  clearMeterReading(): MeterReadingRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): MeterReadingRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): MeterReadingRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MeterReadingRecord.AsObject;
  static toObject(includeInstance: boolean, msg: MeterReadingRecord): MeterReadingRecord.AsObject;
  static serializeBinaryToWriter(message: MeterReadingRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MeterReadingRecord;
  static deserializeBinaryFromReader(message: MeterReadingRecord, reader: jspb.BinaryReader): MeterReadingRecord;
}

export namespace MeterReadingRecord {
  export type AsObject = {
    meterReading?: meter_pb.MeterReading.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ListMeterReadingHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListMeterReadingHistoryRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListMeterReadingHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListMeterReadingHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListMeterReadingHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListMeterReadingHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListMeterReadingHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListMeterReadingHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListMeterReadingHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListMeterReadingHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListMeterReadingHistoryRequest): ListMeterReadingHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListMeterReadingHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListMeterReadingHistoryRequest;
  static deserializeBinaryFromReader(message: ListMeterReadingHistoryRequest, reader: jspb.BinaryReader): ListMeterReadingHistoryRequest;
}

export namespace ListMeterReadingHistoryRequest {
  export type AsObject = {
    name: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListMeterReadingHistoryResponse extends jspb.Message {
  getMeterReadingRecordsList(): Array<MeterReadingRecord>;
  setMeterReadingRecordsList(value: Array<MeterReadingRecord>): ListMeterReadingHistoryResponse;
  clearMeterReadingRecordsList(): ListMeterReadingHistoryResponse;
  addMeterReadingRecords(value?: MeterReadingRecord, index?: number): MeterReadingRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListMeterReadingHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListMeterReadingHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListMeterReadingHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListMeterReadingHistoryResponse): ListMeterReadingHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListMeterReadingHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListMeterReadingHistoryResponse;
  static deserializeBinaryFromReader(message: ListMeterReadingHistoryResponse, reader: jspb.BinaryReader): ListMeterReadingHistoryResponse;
}

export namespace ListMeterReadingHistoryResponse {
  export type AsObject = {
    meterReadingRecordsList: Array<MeterReadingRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

