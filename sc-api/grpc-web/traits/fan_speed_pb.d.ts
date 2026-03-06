import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"


export class FanSpeed extends jspb.Message {
  getPercentage(): number;
  setPercentage(value: number): FanSpeed;

  getPreset(): string;
  setPreset(value: string): FanSpeed;

  getPresetIndex(): number;
  setPresetIndex(value: number): FanSpeed;

  getDirection(): FanSpeed.Direction;
  setDirection(value: FanSpeed.Direction): FanSpeed;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): FanSpeed.AsObject;
  static toObject(includeInstance: boolean, msg: FanSpeed): FanSpeed.AsObject;
  static serializeBinaryToWriter(message: FanSpeed, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): FanSpeed;
  static deserializeBinaryFromReader(message: FanSpeed, reader: jspb.BinaryReader): FanSpeed;
}

export namespace FanSpeed {
  export type AsObject = {
    percentage: number;
    preset: string;
    presetIndex: number;
    direction: FanSpeed.Direction;
  };

  export enum Direction {
    DIRECTION_UNSPECIFIED = 0,
    FORWARD = 1,
    BACKWARD = 2,
  }
}

export class FanSpeedSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): FanSpeedSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): FanSpeedSupport;

  getPresetsList(): Array<string>;
  setPresetsList(value: Array<string>): FanSpeedSupport;
  clearPresetsList(): FanSpeedSupport;
  addPresets(value: string, index?: number): FanSpeedSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): FanSpeedSupport.AsObject;
  static toObject(includeInstance: boolean, msg: FanSpeedSupport): FanSpeedSupport.AsObject;
  static serializeBinaryToWriter(message: FanSpeedSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): FanSpeedSupport;
  static deserializeBinaryFromReader(message: FanSpeedSupport, reader: jspb.BinaryReader): FanSpeedSupport;
}

export namespace FanSpeedSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    presetsList: Array<string>;
  };
}

export class GetFanSpeedRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetFanSpeedRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetFanSpeedRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetFanSpeedRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetFanSpeedRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetFanSpeedRequest): GetFanSpeedRequest.AsObject;
  static serializeBinaryToWriter(message: GetFanSpeedRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetFanSpeedRequest;
  static deserializeBinaryFromReader(message: GetFanSpeedRequest, reader: jspb.BinaryReader): GetFanSpeedRequest;
}

export namespace GetFanSpeedRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateFanSpeedRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateFanSpeedRequest;

  getFanSpeed(): FanSpeed | undefined;
  setFanSpeed(value?: FanSpeed): UpdateFanSpeedRequest;
  hasFanSpeed(): boolean;
  clearFanSpeed(): UpdateFanSpeedRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateFanSpeedRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateFanSpeedRequest;

  getRelative(): boolean;
  setRelative(value: boolean): UpdateFanSpeedRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateFanSpeedRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateFanSpeedRequest): UpdateFanSpeedRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateFanSpeedRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateFanSpeedRequest;
  static deserializeBinaryFromReader(message: UpdateFanSpeedRequest, reader: jspb.BinaryReader): UpdateFanSpeedRequest;
}

export namespace UpdateFanSpeedRequest {
  export type AsObject = {
    name: string;
    fanSpeed?: FanSpeed.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    relative: boolean;
  };
}

export class ReverseFanSpeedDirectionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ReverseFanSpeedDirectionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ReverseFanSpeedDirectionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ReverseFanSpeedDirectionRequest): ReverseFanSpeedDirectionRequest.AsObject;
  static serializeBinaryToWriter(message: ReverseFanSpeedDirectionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ReverseFanSpeedDirectionRequest;
  static deserializeBinaryFromReader(message: ReverseFanSpeedDirectionRequest, reader: jspb.BinaryReader): ReverseFanSpeedDirectionRequest;
}

export namespace ReverseFanSpeedDirectionRequest {
  export type AsObject = {
    name: string;
  };
}

export class PullFanSpeedRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullFanSpeedRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullFanSpeedRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullFanSpeedRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullFanSpeedRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullFanSpeedRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullFanSpeedRequest): PullFanSpeedRequest.AsObject;
  static serializeBinaryToWriter(message: PullFanSpeedRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullFanSpeedRequest;
  static deserializeBinaryFromReader(message: PullFanSpeedRequest, reader: jspb.BinaryReader): PullFanSpeedRequest;
}

export namespace PullFanSpeedRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullFanSpeedResponse extends jspb.Message {
  getChangesList(): Array<PullFanSpeedResponse.Change>;
  setChangesList(value: Array<PullFanSpeedResponse.Change>): PullFanSpeedResponse;
  clearChangesList(): PullFanSpeedResponse;
  addChanges(value?: PullFanSpeedResponse.Change, index?: number): PullFanSpeedResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullFanSpeedResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullFanSpeedResponse): PullFanSpeedResponse.AsObject;
  static serializeBinaryToWriter(message: PullFanSpeedResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullFanSpeedResponse;
  static deserializeBinaryFromReader(message: PullFanSpeedResponse, reader: jspb.BinaryReader): PullFanSpeedResponse;
}

export namespace PullFanSpeedResponse {
  export type AsObject = {
    changesList: Array<PullFanSpeedResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getFanSpeed(): FanSpeed | undefined;
    setFanSpeed(value?: FanSpeed): Change;
    hasFanSpeed(): boolean;
    clearFanSpeed(): Change;

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
      fanSpeed?: FanSpeed.AsObject;
    };
  }

}

export class DescribeFanSpeedRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeFanSpeedRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeFanSpeedRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeFanSpeedRequest): DescribeFanSpeedRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeFanSpeedRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeFanSpeedRequest;
  static deserializeBinaryFromReader(message: DescribeFanSpeedRequest, reader: jspb.BinaryReader): DescribeFanSpeedRequest;
}

export namespace DescribeFanSpeedRequest {
  export type AsObject = {
    name: string;
  };
}

