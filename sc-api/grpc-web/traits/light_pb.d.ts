import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"
import * as types_number_pb from '../types/number_pb'; // proto import: "types/number.proto"
import * as types_tween_pb from '../types/tween_pb'; // proto import: "types/tween.proto"


export class Brightness extends jspb.Message {
  getLevelPercent(): number;
  setLevelPercent(value: number): Brightness;

  getPreset(): LightPreset | undefined;
  setPreset(value?: LightPreset): Brightness;
  hasPreset(): boolean;
  clearPreset(): Brightness;

  getBrightnessTween(): types_tween_pb.Tween | undefined;
  setBrightnessTween(value?: types_tween_pb.Tween): Brightness;
  hasBrightnessTween(): boolean;
  clearBrightnessTween(): Brightness;

  getTargetLevelPercent(): number;
  setTargetLevelPercent(value: number): Brightness;

  getTargetPreset(): LightPreset | undefined;
  setTargetPreset(value?: LightPreset): Brightness;
  hasTargetPreset(): boolean;
  clearTargetPreset(): Brightness;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Brightness.AsObject;
  static toObject(includeInstance: boolean, msg: Brightness): Brightness.AsObject;
  static serializeBinaryToWriter(message: Brightness, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Brightness;
  static deserializeBinaryFromReader(message: Brightness, reader: jspb.BinaryReader): Brightness;
}

export namespace Brightness {
  export type AsObject = {
    levelPercent: number;
    preset?: LightPreset.AsObject;
    brightnessTween?: types_tween_pb.Tween.AsObject;
    targetLevelPercent: number;
    targetPreset?: LightPreset.AsObject;
  };
}

export class LightPreset extends jspb.Message {
  getName(): string;
  setName(value: string): LightPreset;

  getTitle(): string;
  setTitle(value: string): LightPreset;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LightPreset.AsObject;
  static toObject(includeInstance: boolean, msg: LightPreset): LightPreset.AsObject;
  static serializeBinaryToWriter(message: LightPreset, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LightPreset;
  static deserializeBinaryFromReader(message: LightPreset, reader: jspb.BinaryReader): LightPreset;
}

export namespace LightPreset {
  export type AsObject = {
    name: string;
    title: string;
  };
}

export class BrightnessSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): BrightnessSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): BrightnessSupport;

  getBrightnessAttributes(): types_number_pb.Int32Attributes | undefined;
  setBrightnessAttributes(value?: types_number_pb.Int32Attributes): BrightnessSupport;
  hasBrightnessAttributes(): boolean;
  clearBrightnessAttributes(): BrightnessSupport;

  getPresetsList(): Array<LightPreset>;
  setPresetsList(value: Array<LightPreset>): BrightnessSupport;
  clearPresetsList(): BrightnessSupport;
  addPresets(value?: LightPreset, index?: number): LightPreset;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BrightnessSupport.AsObject;
  static toObject(includeInstance: boolean, msg: BrightnessSupport): BrightnessSupport.AsObject;
  static serializeBinaryToWriter(message: BrightnessSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BrightnessSupport;
  static deserializeBinaryFromReader(message: BrightnessSupport, reader: jspb.BinaryReader): BrightnessSupport;
}

export namespace BrightnessSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    brightnessAttributes?: types_number_pb.Int32Attributes.AsObject;
    presetsList: Array<LightPreset.AsObject>;
  };
}

export class UpdateBrightnessRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateBrightnessRequest;

  getBrightness(): Brightness | undefined;
  setBrightness(value?: Brightness): UpdateBrightnessRequest;
  hasBrightness(): boolean;
  clearBrightness(): UpdateBrightnessRequest;

  getDelta(): boolean;
  setDelta(value: boolean): UpdateBrightnessRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateBrightnessRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateBrightnessRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateBrightnessRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateBrightnessRequest): UpdateBrightnessRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateBrightnessRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateBrightnessRequest;
  static deserializeBinaryFromReader(message: UpdateBrightnessRequest, reader: jspb.BinaryReader): UpdateBrightnessRequest;
}

export namespace UpdateBrightnessRequest {
  export type AsObject = {
    name: string;
    brightness?: Brightness.AsObject;
    delta: boolean;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class GetBrightnessRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetBrightnessRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetBrightnessRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetBrightnessRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBrightnessRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetBrightnessRequest): GetBrightnessRequest.AsObject;
  static serializeBinaryToWriter(message: GetBrightnessRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBrightnessRequest;
  static deserializeBinaryFromReader(message: GetBrightnessRequest, reader: jspb.BinaryReader): GetBrightnessRequest;
}

export namespace GetBrightnessRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullBrightnessRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullBrightnessRequest;

  getExcludeRamping(): boolean;
  setExcludeRamping(value: boolean): PullBrightnessRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullBrightnessRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullBrightnessRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullBrightnessRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullBrightnessRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullBrightnessRequest): PullBrightnessRequest.AsObject;
  static serializeBinaryToWriter(message: PullBrightnessRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullBrightnessRequest;
  static deserializeBinaryFromReader(message: PullBrightnessRequest, reader: jspb.BinaryReader): PullBrightnessRequest;
}

export namespace PullBrightnessRequest {
  export type AsObject = {
    name: string;
    excludeRamping: boolean;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullBrightnessResponse extends jspb.Message {
  getChangesList(): Array<PullBrightnessResponse.Change>;
  setChangesList(value: Array<PullBrightnessResponse.Change>): PullBrightnessResponse;
  clearChangesList(): PullBrightnessResponse;
  addChanges(value?: PullBrightnessResponse.Change, index?: number): PullBrightnessResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullBrightnessResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullBrightnessResponse): PullBrightnessResponse.AsObject;
  static serializeBinaryToWriter(message: PullBrightnessResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullBrightnessResponse;
  static deserializeBinaryFromReader(message: PullBrightnessResponse, reader: jspb.BinaryReader): PullBrightnessResponse;
}

export namespace PullBrightnessResponse {
  export type AsObject = {
    changesList: Array<PullBrightnessResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getBrightness(): Brightness | undefined;
    setBrightness(value?: Brightness): Change;
    hasBrightness(): boolean;
    clearBrightness(): Change;

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
      brightness?: Brightness.AsObject;
    };
  }

}

export class DescribeBrightnessRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeBrightnessRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeBrightnessRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeBrightnessRequest): DescribeBrightnessRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeBrightnessRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeBrightnessRequest;
  static deserializeBinaryFromReader(message: DescribeBrightnessRequest, reader: jspb.BinaryReader): DescribeBrightnessRequest;
}

export namespace DescribeBrightnessRequest {
  export type AsObject = {
    name: string;
  };
}

