import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"
import * as traits_air_quality_sensor_pb from '@smart-core-os/sc-api-grpc-web/traits/air_quality_sensor_pb'; // proto import: "traits/air_quality_sensor.proto"


export class AirQualityRecord extends jspb.Message {
  getAirQuality(): traits_air_quality_sensor_pb.AirQuality | undefined;
  setAirQuality(value?: traits_air_quality_sensor_pb.AirQuality): AirQualityRecord;
  hasAirQuality(): boolean;
  clearAirQuality(): AirQualityRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): AirQualityRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): AirQualityRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AirQualityRecord.AsObject;
  static toObject(includeInstance: boolean, msg: AirQualityRecord): AirQualityRecord.AsObject;
  static serializeBinaryToWriter(message: AirQualityRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AirQualityRecord;
  static deserializeBinaryFromReader(message: AirQualityRecord, reader: jspb.BinaryReader): AirQualityRecord;
}

export namespace AirQualityRecord {
  export type AsObject = {
    airQuality?: traits_air_quality_sensor_pb.AirQuality.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ListAirQualityHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListAirQualityHistoryRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListAirQualityHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListAirQualityHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListAirQualityHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListAirQualityHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListAirQualityHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListAirQualityHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListAirQualityHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAirQualityHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListAirQualityHistoryRequest): ListAirQualityHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListAirQualityHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAirQualityHistoryRequest;
  static deserializeBinaryFromReader(message: ListAirQualityHistoryRequest, reader: jspb.BinaryReader): ListAirQualityHistoryRequest;
}

export namespace ListAirQualityHistoryRequest {
  export type AsObject = {
    name: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListAirQualityHistoryResponse extends jspb.Message {
  getAirQualityRecordsList(): Array<AirQualityRecord>;
  setAirQualityRecordsList(value: Array<AirQualityRecord>): ListAirQualityHistoryResponse;
  clearAirQualityRecordsList(): ListAirQualityHistoryResponse;
  addAirQualityRecords(value?: AirQualityRecord, index?: number): AirQualityRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListAirQualityHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListAirQualityHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAirQualityHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListAirQualityHistoryResponse): ListAirQualityHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListAirQualityHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAirQualityHistoryResponse;
  static deserializeBinaryFromReader(message: ListAirQualityHistoryResponse, reader: jspb.BinaryReader): ListAirQualityHistoryResponse;
}

export namespace ListAirQualityHistoryResponse {
  export type AsObject = {
    airQualityRecordsList: Array<AirQualityRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

