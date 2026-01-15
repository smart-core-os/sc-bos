import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as traits_air_temperature_pb from '@smart-core-os/sc-api-grpc-web/traits/air_temperature_pb'; // proto import: "traits/air_temperature.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"


export class AirTemperatureRecord extends jspb.Message {
  getAirTemperature(): traits_air_temperature_pb.AirTemperature | undefined;
  setAirTemperature(value?: traits_air_temperature_pb.AirTemperature): AirTemperatureRecord;
  hasAirTemperature(): boolean;
  clearAirTemperature(): AirTemperatureRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): AirTemperatureRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): AirTemperatureRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AirTemperatureRecord.AsObject;
  static toObject(includeInstance: boolean, msg: AirTemperatureRecord): AirTemperatureRecord.AsObject;
  static serializeBinaryToWriter(message: AirTemperatureRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AirTemperatureRecord;
  static deserializeBinaryFromReader(message: AirTemperatureRecord, reader: jspb.BinaryReader): AirTemperatureRecord;
}

export namespace AirTemperatureRecord {
  export type AsObject = {
    airTemperature?: traits_air_temperature_pb.AirTemperature.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ListAirTemperatureHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListAirTemperatureHistoryRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListAirTemperatureHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListAirTemperatureHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListAirTemperatureHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListAirTemperatureHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListAirTemperatureHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListAirTemperatureHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListAirTemperatureHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAirTemperatureHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListAirTemperatureHistoryRequest): ListAirTemperatureHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListAirTemperatureHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAirTemperatureHistoryRequest;
  static deserializeBinaryFromReader(message: ListAirTemperatureHistoryRequest, reader: jspb.BinaryReader): ListAirTemperatureHistoryRequest;
}

export namespace ListAirTemperatureHistoryRequest {
  export type AsObject = {
    name: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListAirTemperatureHistoryResponse extends jspb.Message {
  getAirTemperatureRecordsList(): Array<AirTemperatureRecord>;
  setAirTemperatureRecordsList(value: Array<AirTemperatureRecord>): ListAirTemperatureHistoryResponse;
  clearAirTemperatureRecordsList(): ListAirTemperatureHistoryResponse;
  addAirTemperatureRecords(value?: AirTemperatureRecord, index?: number): AirTemperatureRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListAirTemperatureHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListAirTemperatureHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAirTemperatureHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListAirTemperatureHistoryResponse): ListAirTemperatureHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListAirTemperatureHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAirTemperatureHistoryResponse;
  static deserializeBinaryFromReader(message: ListAirTemperatureHistoryResponse, reader: jspb.BinaryReader): ListAirTemperatureHistoryResponse;
}

export namespace ListAirTemperatureHistoryResponse {
  export type AsObject = {
    airTemperatureRecordsList: Array<AirTemperatureRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

