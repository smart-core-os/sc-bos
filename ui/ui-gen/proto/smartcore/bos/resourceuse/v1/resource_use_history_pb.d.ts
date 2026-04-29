import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_resourceuse_v1_resource_use_pb from '../../../../smartcore/bos/resourceuse/v1/resource_use_pb'; // proto import: "smartcore/bos/resourceuse/v1/resource_use.proto"
import * as smartcore_bos_types_time_v1_period_pb from '../../../../smartcore/bos/types/time/v1/period_pb'; // proto import: "smartcore/bos/types/time/v1/period.proto"


export class ResourceUseRecord extends jspb.Message {
  getResourceUse(): smartcore_bos_resourceuse_v1_resource_use_pb.ResourceUse | undefined;
  setResourceUse(value?: smartcore_bos_resourceuse_v1_resource_use_pb.ResourceUse): ResourceUseRecord;
  hasResourceUse(): boolean;
  clearResourceUse(): ResourceUseRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): ResourceUseRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): ResourceUseRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ResourceUseRecord.AsObject;
  static toObject(includeInstance: boolean, msg: ResourceUseRecord): ResourceUseRecord.AsObject;
  static serializeBinaryToWriter(message: ResourceUseRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ResourceUseRecord;
  static deserializeBinaryFromReader(message: ResourceUseRecord, reader: jspb.BinaryReader): ResourceUseRecord;
}

export namespace ResourceUseRecord {
  export type AsObject = {
    resourceUse?: smartcore_bos_resourceuse_v1_resource_use_pb.ResourceUse.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ListResourceUseHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListResourceUseHistoryRequest;

  getPeriod(): smartcore_bos_types_time_v1_period_pb.Period | undefined;
  setPeriod(value?: smartcore_bos_types_time_v1_period_pb.Period): ListResourceUseHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListResourceUseHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListResourceUseHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListResourceUseHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListResourceUseHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListResourceUseHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListResourceUseHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListResourceUseHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListResourceUseHistoryRequest): ListResourceUseHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListResourceUseHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListResourceUseHistoryRequest;
  static deserializeBinaryFromReader(message: ListResourceUseHistoryRequest, reader: jspb.BinaryReader): ListResourceUseHistoryRequest;
}

export namespace ListResourceUseHistoryRequest {
  export type AsObject = {
    name: string;
    period?: smartcore_bos_types_time_v1_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListResourceUseHistoryResponse extends jspb.Message {
  getResourceUseRecordsList(): Array<ResourceUseRecord>;
  setResourceUseRecordsList(value: Array<ResourceUseRecord>): ListResourceUseHistoryResponse;
  clearResourceUseRecordsList(): ListResourceUseHistoryResponse;
  addResourceUseRecords(value?: ResourceUseRecord, index?: number): ResourceUseRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListResourceUseHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListResourceUseHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListResourceUseHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListResourceUseHistoryResponse): ListResourceUseHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListResourceUseHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListResourceUseHistoryResponse;
  static deserializeBinaryFromReader(message: ListResourceUseHistoryResponse, reader: jspb.BinaryReader): ListResourceUseHistoryResponse;
}

export namespace ListResourceUseHistoryResponse {
  export type AsObject = {
    resourceUseRecordsList: Array<ResourceUseRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

