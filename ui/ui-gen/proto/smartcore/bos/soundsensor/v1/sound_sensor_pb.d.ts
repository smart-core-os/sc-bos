import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '@smart-core-os/sc-api-grpc-web/types/info_pb'; // proto import: "types/info.proto"


export class SoundLevel extends jspb.Message {
  getSoundPressureLevel(): number;
  setSoundPressureLevel(value: number): SoundLevel;
  hasSoundPressureLevel(): boolean;
  clearSoundPressureLevel(): SoundLevel;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SoundLevel.AsObject;
  static toObject(includeInstance: boolean, msg: SoundLevel): SoundLevel.AsObject;
  static serializeBinaryToWriter(message: SoundLevel, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SoundLevel;
  static deserializeBinaryFromReader(message: SoundLevel, reader: jspb.BinaryReader): SoundLevel;
}

export namespace SoundLevel {
  export type AsObject = {
    soundPressureLevel?: number;
  };

  export enum SoundPressureLevelCase {
    _SOUND_PRESSURE_LEVEL_NOT_SET = 0,
    SOUND_PRESSURE_LEVEL = 1,
  }
}

export class SoundLevelSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): SoundLevelSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): SoundLevelSupport;

  getSoundLevelUnit(): string;
  setSoundLevelUnit(value: string): SoundLevelSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SoundLevelSupport.AsObject;
  static toObject(includeInstance: boolean, msg: SoundLevelSupport): SoundLevelSupport.AsObject;
  static serializeBinaryToWriter(message: SoundLevelSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SoundLevelSupport;
  static deserializeBinaryFromReader(message: SoundLevelSupport, reader: jspb.BinaryReader): SoundLevelSupport;
}

export namespace SoundLevelSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    soundLevelUnit: string;
  };
}

export class GetSoundLevelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetSoundLevelRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetSoundLevelRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetSoundLevelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetSoundLevelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetSoundLevelRequest): GetSoundLevelRequest.AsObject;
  static serializeBinaryToWriter(message: GetSoundLevelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetSoundLevelRequest;
  static deserializeBinaryFromReader(message: GetSoundLevelRequest, reader: jspb.BinaryReader): GetSoundLevelRequest;
}

export namespace GetSoundLevelRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullSoundLevelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullSoundLevelRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullSoundLevelRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullSoundLevelRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullSoundLevelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullSoundLevelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullSoundLevelRequest): PullSoundLevelRequest.AsObject;
  static serializeBinaryToWriter(message: PullSoundLevelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullSoundLevelRequest;
  static deserializeBinaryFromReader(message: PullSoundLevelRequest, reader: jspb.BinaryReader): PullSoundLevelRequest;
}

export namespace PullSoundLevelRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullSoundLevelResponse extends jspb.Message {
  getChangesList(): Array<PullSoundLevelResponse.Change>;
  setChangesList(value: Array<PullSoundLevelResponse.Change>): PullSoundLevelResponse;
  clearChangesList(): PullSoundLevelResponse;
  addChanges(value?: PullSoundLevelResponse.Change, index?: number): PullSoundLevelResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullSoundLevelResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullSoundLevelResponse): PullSoundLevelResponse.AsObject;
  static serializeBinaryToWriter(message: PullSoundLevelResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullSoundLevelResponse;
  static deserializeBinaryFromReader(message: PullSoundLevelResponse, reader: jspb.BinaryReader): PullSoundLevelResponse;
}

export namespace PullSoundLevelResponse {
  export type AsObject = {
    changesList: Array<PullSoundLevelResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getSoundLevel(): SoundLevel | undefined;
    setSoundLevel(value?: SoundLevel): Change;
    hasSoundLevel(): boolean;
    clearSoundLevel(): Change;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Change.AsObject;
    static toObject(includeInstance: boolean, msg: Change): Change.AsObject;
    static serializeBinaryToWriter(message: Change, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Change;
    static deserializeBinaryFromReader(message: Change, reader: jspb.BinaryReader): Change;
  }

  export namespace Change {
    export type AsObject = {
      name: string;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      soundLevel?: SoundLevel.AsObject;
    };
  }

}

export class DescribeSoundLevelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeSoundLevelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeSoundLevelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeSoundLevelRequest): DescribeSoundLevelRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeSoundLevelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeSoundLevelRequest;
  static deserializeBinaryFromReader(message: DescribeSoundLevelRequest, reader: jspb.BinaryReader): DescribeSoundLevelRequest;
}

export namespace DescribeSoundLevelRequest {
  export type AsObject = {
    name: string;
  };
}

