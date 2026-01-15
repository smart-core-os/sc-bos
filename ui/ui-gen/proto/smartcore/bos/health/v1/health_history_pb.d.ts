import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_health_v1_health_pb from '../../../../smartcore/bos/health/v1/health_pb'; // proto import: "smartcore/bos/health/v1/health.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"


export class HealthCheckRecord extends jspb.Message {
  getHealthCheck(): smartcore_bos_health_v1_health_pb.HealthCheck | undefined;
  setHealthCheck(value?: smartcore_bos_health_v1_health_pb.HealthCheck): HealthCheckRecord;
  hasHealthCheck(): boolean;
  clearHealthCheck(): HealthCheckRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): HealthCheckRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): HealthCheckRecord;

  getRecordType(): HealthCheckRecord.RecordType;
  setRecordType(value: HealthCheckRecord.RecordType): HealthCheckRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): HealthCheckRecord.AsObject;
  static toObject(includeInstance: boolean, msg: HealthCheckRecord): HealthCheckRecord.AsObject;
  static serializeBinaryToWriter(message: HealthCheckRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): HealthCheckRecord;
  static deserializeBinaryFromReader(message: HealthCheckRecord, reader: jspb.BinaryReader): HealthCheckRecord;
}

export namespace HealthCheckRecord {
  export type AsObject = {
    healthCheck?: smartcore_bos_health_v1_health_pb.HealthCheck.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    recordType: HealthCheckRecord.RecordType;
  };

  export enum RecordType {
    RECORD_TYPE_UNSPECIFIED = 0,
    ADDED = 1,
    UPDATED = 2,
    REMOVED = 3,
  }
}

export class ListHealthCheckHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListHealthCheckHistoryRequest;

  getId(): string;
  setId(value: string): ListHealthCheckHistoryRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListHealthCheckHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListHealthCheckHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListHealthCheckHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListHealthCheckHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListHealthCheckHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListHealthCheckHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListHealthCheckHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListHealthCheckHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListHealthCheckHistoryRequest): ListHealthCheckHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListHealthCheckHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListHealthCheckHistoryRequest;
  static deserializeBinaryFromReader(message: ListHealthCheckHistoryRequest, reader: jspb.BinaryReader): ListHealthCheckHistoryRequest;
}

export namespace ListHealthCheckHistoryRequest {
  export type AsObject = {
    name: string;
    id: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListHealthCheckHistoryResponse extends jspb.Message {
  getHealthCheckRecordsList(): Array<HealthCheckRecord>;
  setHealthCheckRecordsList(value: Array<HealthCheckRecord>): ListHealthCheckHistoryResponse;
  clearHealthCheckRecordsList(): ListHealthCheckHistoryResponse;
  addHealthCheckRecords(value?: HealthCheckRecord, index?: number): HealthCheckRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListHealthCheckHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListHealthCheckHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListHealthCheckHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListHealthCheckHistoryResponse): ListHealthCheckHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListHealthCheckHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListHealthCheckHistoryResponse;
  static deserializeBinaryFromReader(message: ListHealthCheckHistoryResponse, reader: jspb.BinaryReader): ListHealthCheckHistoryResponse;
}

export namespace ListHealthCheckHistoryResponse {
  export type AsObject = {
    healthCheckRecordsList: Array<HealthCheckRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

