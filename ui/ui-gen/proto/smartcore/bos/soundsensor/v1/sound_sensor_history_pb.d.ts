import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_soundsensor_v1_sound_sensor_pb from '../../../../smartcore/bos/soundsensor/v1/sound_sensor_pb'; // proto import: "smartcore/bos/soundsensor/v1/sound_sensor.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"


export class SoundLevelRecord extends jspb.Message {
  getSoundLevel(): smartcore_bos_soundsensor_v1_sound_sensor_pb.SoundLevel | undefined;
  setSoundLevel(value?: smartcore_bos_soundsensor_v1_sound_sensor_pb.SoundLevel): SoundLevelRecord;
  hasSoundLevel(): boolean;
  clearSoundLevel(): SoundLevelRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): SoundLevelRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): SoundLevelRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SoundLevelRecord.AsObject;
  static toObject(includeInstance: boolean, msg: SoundLevelRecord): SoundLevelRecord.AsObject;
  static serializeBinaryToWriter(message: SoundLevelRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SoundLevelRecord;
  static deserializeBinaryFromReader(message: SoundLevelRecord, reader: jspb.BinaryReader): SoundLevelRecord;
}

export namespace SoundLevelRecord {
  export type AsObject = {
    soundLevel?: smartcore_bos_soundsensor_v1_sound_sensor_pb.SoundLevel.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ListSoundLevelHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListSoundLevelHistoryRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListSoundLevelHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListSoundLevelHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListSoundLevelHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListSoundLevelHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListSoundLevelHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListSoundLevelHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListSoundLevelHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListSoundLevelHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListSoundLevelHistoryRequest): ListSoundLevelHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListSoundLevelHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListSoundLevelHistoryRequest;
  static deserializeBinaryFromReader(message: ListSoundLevelHistoryRequest, reader: jspb.BinaryReader): ListSoundLevelHistoryRequest;
}

export namespace ListSoundLevelHistoryRequest {
  export type AsObject = {
    name: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListSoundLevelHistoryResponse extends jspb.Message {
  getSoundLevelRecordsList(): Array<SoundLevelRecord>;
  setSoundLevelRecordsList(value: Array<SoundLevelRecord>): ListSoundLevelHistoryResponse;
  clearSoundLevelRecordsList(): ListSoundLevelHistoryResponse;
  addSoundLevelRecords(value?: SoundLevelRecord, index?: number): SoundLevelRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListSoundLevelHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListSoundLevelHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListSoundLevelHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListSoundLevelHistoryResponse): ListSoundLevelHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListSoundLevelHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListSoundLevelHistoryResponse;
  static deserializeBinaryFromReader(message: ListSoundLevelHistoryResponse, reader: jspb.BinaryReader): ListSoundLevelHistoryResponse;
}

export namespace ListSoundLevelHistoryResponse {
  export type AsObject = {
    soundLevelRecordsList: Array<SoundLevelRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

