import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_resourceutilisation_v1_resource_utilisation_pb from '../../../../smartcore/bos/resourceutilisation/v1/resource_utilisation_pb'; // proto import: "smartcore/bos/resourceutilisation/v1/resource_utilisation.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"


export class ResourceUtilisationRecord extends jspb.Message {
  getResourceUtilisation(): smartcore_bos_resourceutilisation_v1_resource_utilisation_pb.ResourceUtilisation | undefined;
  setResourceUtilisation(value?: smartcore_bos_resourceutilisation_v1_resource_utilisation_pb.ResourceUtilisation): ResourceUtilisationRecord;
  hasResourceUtilisation(): boolean;
  clearResourceUtilisation(): ResourceUtilisationRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): ResourceUtilisationRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): ResourceUtilisationRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ResourceUtilisationRecord.AsObject;
  static toObject(includeInstance: boolean, msg: ResourceUtilisationRecord): ResourceUtilisationRecord.AsObject;
  static serializeBinaryToWriter(message: ResourceUtilisationRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ResourceUtilisationRecord;
  static deserializeBinaryFromReader(message: ResourceUtilisationRecord, reader: jspb.BinaryReader): ResourceUtilisationRecord;
}

export namespace ResourceUtilisationRecord {
  export type AsObject = {
    resourceUtilisation?: smartcore_bos_resourceutilisation_v1_resource_utilisation_pb.ResourceUtilisation.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ListResourceUtilisationHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListResourceUtilisationHistoryRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListResourceUtilisationHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListResourceUtilisationHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListResourceUtilisationHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListResourceUtilisationHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListResourceUtilisationHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListResourceUtilisationHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListResourceUtilisationHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListResourceUtilisationHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListResourceUtilisationHistoryRequest): ListResourceUtilisationHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListResourceUtilisationHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListResourceUtilisationHistoryRequest;
  static deserializeBinaryFromReader(message: ListResourceUtilisationHistoryRequest, reader: jspb.BinaryReader): ListResourceUtilisationHistoryRequest;
}

export namespace ListResourceUtilisationHistoryRequest {
  export type AsObject = {
    name: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListResourceUtilisationHistoryResponse extends jspb.Message {
  getResourceUtilisationRecordsList(): Array<ResourceUtilisationRecord>;
  setResourceUtilisationRecordsList(value: Array<ResourceUtilisationRecord>): ListResourceUtilisationHistoryResponse;
  clearResourceUtilisationRecordsList(): ListResourceUtilisationHistoryResponse;
  addResourceUtilisationRecords(value?: ResourceUtilisationRecord, index?: number): ResourceUtilisationRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListResourceUtilisationHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListResourceUtilisationHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListResourceUtilisationHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListResourceUtilisationHistoryResponse): ListResourceUtilisationHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListResourceUtilisationHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListResourceUtilisationHistoryResponse;
  static deserializeBinaryFromReader(message: ListResourceUtilisationHistoryResponse, reader: jspb.BinaryReader): ListResourceUtilisationHistoryResponse;
}

export namespace ListResourceUtilisationHistoryResponse {
  export type AsObject = {
    resourceUtilisationRecordsList: Array<ResourceUtilisationRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

