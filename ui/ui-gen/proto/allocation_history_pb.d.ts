import * as jspb from 'google-protobuf'

import * as allocation_pb from './allocation_pb'; // proto import: "allocation.proto"
import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"


export class AllocationRecord extends jspb.Message {
  getAllocation(): allocation_pb.Allocation | undefined;
  setAllocation(value?: allocation_pb.Allocation): AllocationRecord;
  hasAllocation(): boolean;
  clearAllocation(): AllocationRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): AllocationRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): AllocationRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AllocationRecord.AsObject;
  static toObject(includeInstance: boolean, msg: AllocationRecord): AllocationRecord.AsObject;
  static serializeBinaryToWriter(message: AllocationRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AllocationRecord;
  static deserializeBinaryFromReader(message: AllocationRecord, reader: jspb.BinaryReader): AllocationRecord;
}

export namespace AllocationRecord {
  export type AsObject = {
    allocation?: allocation_pb.Allocation.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ListAllocationHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListAllocationHistoryRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListAllocationHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListAllocationHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListAllocationHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListAllocationHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListAllocationHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListAllocationHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListAllocationHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAllocationHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListAllocationHistoryRequest): ListAllocationHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListAllocationHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAllocationHistoryRequest;
  static deserializeBinaryFromReader(message: ListAllocationHistoryRequest, reader: jspb.BinaryReader): ListAllocationHistoryRequest;
}

export namespace ListAllocationHistoryRequest {
  export type AsObject = {
    name: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListAllocationHistoryResponse extends jspb.Message {
  getAllocationRecordsList(): Array<AllocationRecord>;
  setAllocationRecordsList(value: Array<AllocationRecord>): ListAllocationHistoryResponse;
  clearAllocationRecordsList(): ListAllocationHistoryResponse;
  addAllocationRecords(value?: AllocationRecord, index?: number): AllocationRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListAllocationHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListAllocationHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAllocationHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListAllocationHistoryResponse): ListAllocationHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListAllocationHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAllocationHistoryResponse;
  static deserializeBinaryFromReader(message: ListAllocationHistoryResponse, reader: jspb.BinaryReader): ListAllocationHistoryResponse;
}

export namespace ListAllocationHistoryResponse {
  export type AsObject = {
    allocationRecordsList: Array<AllocationRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

