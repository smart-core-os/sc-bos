import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_actor_v1_actor_pb from '../../../../smartcore/bos/actor/v1/actor_pb'; // proto import: "smartcore/bos/actor/v1/actor.proto"
import * as smartcore_bos_types_time_v1_period_pb from '../../../../smartcore/bos/types/time/v1/period_pb'; // proto import: "smartcore/bos/types/time/v1/period.proto"


export class BootRecord extends jspb.Message {
  getRebootTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRebootTime(value?: google_protobuf_timestamp_pb.Timestamp): BootRecord;
  hasRebootTime(): boolean;
  clearRebootTime(): BootRecord;

  getReason(): string;
  setReason(value: string): BootRecord;

  getActor(): smartcore_bos_actor_v1_actor_pb.Actor | undefined;
  setActor(value?: smartcore_bos_actor_v1_actor_pb.Actor): BootRecord;
  hasActor(): boolean;
  clearActor(): BootRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): BootRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): BootRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BootRecord.AsObject;
  static toObject(includeInstance: boolean, msg: BootRecord): BootRecord.AsObject;
  static serializeBinaryToWriter(message: BootRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BootRecord;
  static deserializeBinaryFromReader(message: BootRecord, reader: jspb.BinaryReader): BootRecord;
}

export namespace BootRecord {
  export type AsObject = {
    rebootTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    reason: string;
    actor?: smartcore_bos_actor_v1_actor_pb.Actor.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ListBootRecordsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListBootRecordsRequest;

  getPeriod(): smartcore_bos_types_time_v1_period_pb.Period | undefined;
  setPeriod(value?: smartcore_bos_types_time_v1_period_pb.Period): ListBootRecordsRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListBootRecordsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListBootRecordsRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListBootRecordsRequest;

  getPageSize(): number;
  setPageSize(value: number): ListBootRecordsRequest;

  getPageToken(): string;
  setPageToken(value: string): ListBootRecordsRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListBootRecordsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListBootRecordsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListBootRecordsRequest): ListBootRecordsRequest.AsObject;
  static serializeBinaryToWriter(message: ListBootRecordsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListBootRecordsRequest;
  static deserializeBinaryFromReader(message: ListBootRecordsRequest, reader: jspb.BinaryReader): ListBootRecordsRequest;
}

export namespace ListBootRecordsRequest {
  export type AsObject = {
    name: string;
    period?: smartcore_bos_types_time_v1_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListBootRecordsResponse extends jspb.Message {
  getBootRecordsList(): Array<BootRecord>;
  setBootRecordsList(value: Array<BootRecord>): ListBootRecordsResponse;
  clearBootRecordsList(): ListBootRecordsResponse;
  addBootRecords(value?: BootRecord, index?: number): BootRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListBootRecordsResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListBootRecordsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListBootRecordsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListBootRecordsResponse): ListBootRecordsResponse.AsObject;
  static serializeBinaryToWriter(message: ListBootRecordsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListBootRecordsResponse;
  static deserializeBinaryFromReader(message: ListBootRecordsResponse, reader: jspb.BinaryReader): ListBootRecordsResponse;
}

export namespace ListBootRecordsResponse {
  export type AsObject = {
    bootRecordsList: Array<BootRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

