import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"
import * as types_number_pb from '../types/number_pb'; // proto import: "types/number.proto"
import * as types_tween_pb from '../types/tween_pb'; // proto import: "types/tween.proto"


export class Color extends jspb.Message {
  getChannels(): ColorChannels | undefined;
  setChannels(value?: ColorChannels): Color;
  hasChannels(): boolean;
  clearChannels(): Color;

  getPreset(): ColorPreset | undefined;
  setPreset(value?: ColorPreset): Color;
  hasPreset(): boolean;
  clearPreset(): Color;

  getColorTween(): types_tween_pb.Tween | undefined;
  setColorTween(value?: types_tween_pb.Tween): Color;
  hasColorTween(): boolean;
  clearColorTween(): Color;

  getTargetChannels(): ColorChannels | undefined;
  setTargetChannels(value?: ColorChannels): Color;
  hasTargetChannels(): boolean;
  clearTargetChannels(): Color;

  getTargetPreset(): ColorPreset | undefined;
  setTargetPreset(value?: ColorPreset): Color;
  hasTargetPreset(): boolean;
  clearTargetPreset(): Color;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Color.AsObject;
  static toObject(includeInstance: boolean, msg: Color): Color.AsObject;
  static serializeBinaryToWriter(message: Color, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Color;
  static deserializeBinaryFromReader(message: Color, reader: jspb.BinaryReader): Color;
}

export namespace Color {
  export type AsObject = {
    channels?: ColorChannels.AsObject;
    preset?: ColorPreset.AsObject;
    colorTween?: types_tween_pb.Tween.AsObject;
    targetChannels?: ColorChannels.AsObject;
    targetPreset?: ColorPreset.AsObject;
  };
}

export class ColorChannels extends jspb.Message {
  getRed(): number;
  setRed(value: number): ColorChannels;
  hasRed(): boolean;
  clearRed(): ColorChannels;

  getGreen(): number;
  setGreen(value: number): ColorChannels;
  hasGreen(): boolean;
  clearGreen(): ColorChannels;

  getBlue(): number;
  setBlue(value: number): ColorChannels;
  hasBlue(): boolean;
  clearBlue(): ColorChannels;

  getIntensity(): number;
  setIntensity(value: number): ColorChannels;
  hasIntensity(): boolean;
  clearIntensity(): ColorChannels;

  getTemperature(): number;
  setTemperature(value: number): ColorChannels;
  hasTemperature(): boolean;
  clearTemperature(): ColorChannels;

  getMoreMap(): jspb.Map<string, number>;
  clearMoreMap(): ColorChannels;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ColorChannels.AsObject;
  static toObject(includeInstance: boolean, msg: ColorChannels): ColorChannels.AsObject;
  static serializeBinaryToWriter(message: ColorChannels, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ColorChannels;
  static deserializeBinaryFromReader(message: ColorChannels, reader: jspb.BinaryReader): ColorChannels;
}

export namespace ColorChannels {
  export type AsObject = {
    red?: number;
    green?: number;
    blue?: number;
    intensity?: number;
    temperature?: number;
    moreMap: Array<[string, number]>;
  };

  export enum RedCase {
    _RED_NOT_SET = 0,
    RED = 1,
  }

  export enum GreenCase {
    _GREEN_NOT_SET = 0,
    GREEN = 2,
  }

  export enum BlueCase {
    _BLUE_NOT_SET = 0,
    BLUE = 3,
  }

  export enum IntensityCase {
    _INTENSITY_NOT_SET = 0,
    INTENSITY = 4,
  }

  export enum TemperatureCase {
    _TEMPERATURE_NOT_SET = 0,
    TEMPERATURE = 5,
  }
}

export class ColorPreset extends jspb.Message {
  getName(): string;
  setName(value: string): ColorPreset;

  getTitle(): string;
  setTitle(value: string): ColorPreset;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ColorPreset.AsObject;
  static toObject(includeInstance: boolean, msg: ColorPreset): ColorPreset.AsObject;
  static serializeBinaryToWriter(message: ColorPreset, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ColorPreset;
  static deserializeBinaryFromReader(message: ColorPreset, reader: jspb.BinaryReader): ColorPreset;
}

export namespace ColorPreset {
  export type AsObject = {
    name: string;
    title: string;
  };
}

export class ColorSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): ColorSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): ColorSupport;

  getColorAttributes(): types_number_pb.Int32Attributes | undefined;
  setColorAttributes(value?: types_number_pb.Int32Attributes): ColorSupport;
  hasColorAttributes(): boolean;
  clearColorAttributes(): ColorSupport;

  getPresetsList(): Array<ColorPreset>;
  setPresetsList(value: Array<ColorPreset>): ColorSupport;
  clearPresetsList(): ColorSupport;
  addPresets(value?: ColorPreset, index?: number): ColorPreset;

  getChannelsList(): Array<string>;
  setChannelsList(value: Array<string>): ColorSupport;
  clearChannelsList(): ColorSupport;
  addChannels(value: string, index?: number): ColorSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ColorSupport.AsObject;
  static toObject(includeInstance: boolean, msg: ColorSupport): ColorSupport.AsObject;
  static serializeBinaryToWriter(message: ColorSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ColorSupport;
  static deserializeBinaryFromReader(message: ColorSupport, reader: jspb.BinaryReader): ColorSupport;
}

export namespace ColorSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    colorAttributes?: types_number_pb.Int32Attributes.AsObject;
    presetsList: Array<ColorPreset.AsObject>;
    channelsList: Array<string>;
  };
}

export class UpdateColorRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateColorRequest;

  getColor(): Color | undefined;
  setColor(value?: Color): UpdateColorRequest;
  hasColor(): boolean;
  clearColor(): UpdateColorRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateColorRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateColorRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateColorRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateColorRequest): UpdateColorRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateColorRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateColorRequest;
  static deserializeBinaryFromReader(message: UpdateColorRequest, reader: jspb.BinaryReader): UpdateColorRequest;
}

export namespace UpdateColorRequest {
  export type AsObject = {
    name: string;
    color?: Color.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class GetColorRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetColorRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetColorRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetColorRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetColorRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetColorRequest): GetColorRequest.AsObject;
  static serializeBinaryToWriter(message: GetColorRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetColorRequest;
  static deserializeBinaryFromReader(message: GetColorRequest, reader: jspb.BinaryReader): GetColorRequest;
}

export namespace GetColorRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullColorRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullColorRequest;

  getExcludeRamping(): boolean;
  setExcludeRamping(value: boolean): PullColorRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullColorRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullColorRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullColorRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullColorRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullColorRequest): PullColorRequest.AsObject;
  static serializeBinaryToWriter(message: PullColorRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullColorRequest;
  static deserializeBinaryFromReader(message: PullColorRequest, reader: jspb.BinaryReader): PullColorRequest;
}

export namespace PullColorRequest {
  export type AsObject = {
    name: string;
    excludeRamping: boolean;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullColorResponse extends jspb.Message {
  getChangesList(): Array<PullColorResponse.Change>;
  setChangesList(value: Array<PullColorResponse.Change>): PullColorResponse;
  clearChangesList(): PullColorResponse;
  addChanges(value?: PullColorResponse.Change, index?: number): PullColorResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullColorResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullColorResponse): PullColorResponse.AsObject;
  static serializeBinaryToWriter(message: PullColorResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullColorResponse;
  static deserializeBinaryFromReader(message: PullColorResponse, reader: jspb.BinaryReader): PullColorResponse;
}

export namespace PullColorResponse {
  export type AsObject = {
    changesList: Array<PullColorResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getColor(): Color | undefined;
    setColor(value?: Color): Change;
    hasColor(): boolean;
    clearColor(): Change;

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
      color?: Color.AsObject;
    };
  }

}

export class DescribeColorRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeColorRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeColorRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeColorRequest): DescribeColorRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeColorRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeColorRequest;
  static deserializeBinaryFromReader(message: DescribeColorRequest, reader: jspb.BinaryReader): DescribeColorRequest;
}

export namespace DescribeColorRequest {
  export type AsObject = {
    name: string;
  };
}

