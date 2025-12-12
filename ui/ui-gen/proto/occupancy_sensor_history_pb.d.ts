import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"
import * as traits_occupancy_sensor_pb from '@smart-core-os/sc-api-grpc-web/traits/occupancy_sensor_pb'; // proto import: "traits/occupancy_sensor.proto"


export class OccupancyRecord extends jspb.Message {
  getOccupancy(): traits_occupancy_sensor_pb.Occupancy | undefined;
  setOccupancy(value?: traits_occupancy_sensor_pb.Occupancy): OccupancyRecord;
  hasOccupancy(): boolean;
  clearOccupancy(): OccupancyRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): OccupancyRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): OccupancyRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): OccupancyRecord.AsObject;
  static toObject(includeInstance: boolean, msg: OccupancyRecord): OccupancyRecord.AsObject;
  static serializeBinaryToWriter(message: OccupancyRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): OccupancyRecord;
  static deserializeBinaryFromReader(message: OccupancyRecord, reader: jspb.BinaryReader): OccupancyRecord;
}

export namespace OccupancyRecord {
  export type AsObject = {
    occupancy?: traits_occupancy_sensor_pb.Occupancy.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ListOccupancyHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListOccupancyHistoryRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListOccupancyHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListOccupancyHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListOccupancyHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListOccupancyHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListOccupancyHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListOccupancyHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListOccupancyHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListOccupancyHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListOccupancyHistoryRequest): ListOccupancyHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListOccupancyHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListOccupancyHistoryRequest;
  static deserializeBinaryFromReader(message: ListOccupancyHistoryRequest, reader: jspb.BinaryReader): ListOccupancyHistoryRequest;
}

export namespace ListOccupancyHistoryRequest {
  export type AsObject = {
    name: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListOccupancyHistoryResponse extends jspb.Message {
  getOccupancyRecordsList(): Array<OccupancyRecord>;
  setOccupancyRecordsList(value: Array<OccupancyRecord>): ListOccupancyHistoryResponse;
  clearOccupancyRecordsList(): ListOccupancyHistoryResponse;
  addOccupancyRecords(value?: OccupancyRecord, index?: number): OccupancyRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListOccupancyHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListOccupancyHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListOccupancyHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListOccupancyHistoryResponse): ListOccupancyHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListOccupancyHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListOccupancyHistoryResponse;
  static deserializeBinaryFromReader(message: ListOccupancyHistoryResponse, reader: jspb.BinaryReader): ListOccupancyHistoryResponse;
}

export namespace ListOccupancyHistoryResponse {
  export type AsObject = {
    occupancyRecordsList: Array<OccupancyRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

