import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"
import * as types_number_pb from '../types/number_pb'; // proto import: "types/number.proto"
import * as types_tween_pb from '../types/tween_pb'; // proto import: "types/tween.proto"


export class OpenClosePositions extends jspb.Message {
  getStatesList(): Array<OpenClosePosition>;
  setStatesList(value: Array<OpenClosePosition>): OpenClosePositions;
  clearStatesList(): OpenClosePositions;
  addStates(value?: OpenClosePosition, index?: number): OpenClosePosition;

  getPreset(): OpenClosePositions.Preset | undefined;
  setPreset(value?: OpenClosePositions.Preset): OpenClosePositions;
  hasPreset(): boolean;
  clearPreset(): OpenClosePositions;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): OpenClosePositions.AsObject;
  static toObject(includeInstance: boolean, msg: OpenClosePositions): OpenClosePositions.AsObject;
  static serializeBinaryToWriter(message: OpenClosePositions, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): OpenClosePositions;
  static deserializeBinaryFromReader(message: OpenClosePositions, reader: jspb.BinaryReader): OpenClosePositions;
}

export namespace OpenClosePositions {
  export type AsObject = {
    statesList: Array<OpenClosePosition.AsObject>;
    preset?: OpenClosePositions.Preset.AsObject;
  };

  export class Preset extends jspb.Message {
    getName(): string;
    setName(value: string): Preset;

    getTitle(): string;
    setTitle(value: string): Preset;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Preset.AsObject;
    static toObject(includeInstance: boolean, msg: Preset): Preset.AsObject;
    static serializeBinaryToWriter(message: Preset, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Preset;
    static deserializeBinaryFromReader(message: Preset, reader: jspb.BinaryReader): Preset;
  }

  export namespace Preset {
    export type AsObject = {
      name: string;
      title: string;
    };
  }

}

export class OpenClosePosition extends jspb.Message {
  getOpenPercent(): number;
  setOpenPercent(value: number): OpenClosePosition;

  getOpenPercentTween(): types_tween_pb.Tween | undefined;
  setOpenPercentTween(value?: types_tween_pb.Tween): OpenClosePosition;
  hasOpenPercentTween(): boolean;
  clearOpenPercentTween(): OpenClosePosition;

  getTargetOpenPercent(): number;
  setTargetOpenPercent(value: number): OpenClosePosition;

  getDirection(): OpenClosePosition.Direction;
  setDirection(value: OpenClosePosition.Direction): OpenClosePosition;

  getResistance(): OpenClosePosition.Resistance;
  setResistance(value: OpenClosePosition.Resistance): OpenClosePosition;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): OpenClosePosition.AsObject;
  static toObject(includeInstance: boolean, msg: OpenClosePosition): OpenClosePosition.AsObject;
  static serializeBinaryToWriter(message: OpenClosePosition, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): OpenClosePosition;
  static deserializeBinaryFromReader(message: OpenClosePosition, reader: jspb.BinaryReader): OpenClosePosition;
}

export namespace OpenClosePosition {
  export type AsObject = {
    openPercent: number;
    openPercentTween?: types_tween_pb.Tween.AsObject;
    targetOpenPercent: number;
    direction: OpenClosePosition.Direction;
    resistance: OpenClosePosition.Resistance;
  };

  export enum Direction {
    DIRECTION_UNSPECIFIED = 0,
    UP = 1,
    DOWN = 2,
    LEFT = 3,
    RIGHT = 4,
    IN = 5,
    OUT = 6,
  }

  export enum Resistance {
    RESISTANCE_UNSPECIFIED = 0,
    HELD = 1,
    REDUCED_MOTION = 2,
    SLOW = 3,
  }
}

export class PositionsSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): PositionsSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): PositionsSupport;

  getOpenPercentAttributes(): types_number_pb.FloatAttributes | undefined;
  setOpenPercentAttributes(value?: types_number_pb.FloatAttributes): PositionsSupport;
  hasOpenPercentAttributes(): boolean;
  clearOpenPercentAttributes(): PositionsSupport;

  getDirectionsList(): Array<OpenClosePosition.Direction>;
  setDirectionsList(value: Array<OpenClosePosition.Direction>): PositionsSupport;
  clearDirectionsList(): PositionsSupport;
  addDirections(value: OpenClosePosition.Direction, index?: number): PositionsSupport;

  getSupportsStop(): boolean;
  setSupportsStop(value: boolean): PositionsSupport;

  getPresetsList(): Array<OpenClosePositions.Preset>;
  setPresetsList(value: Array<OpenClosePositions.Preset>): PositionsSupport;
  clearPresetsList(): PositionsSupport;
  addPresets(value?: OpenClosePositions.Preset, index?: number): OpenClosePositions.Preset;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PositionsSupport.AsObject;
  static toObject(includeInstance: boolean, msg: PositionsSupport): PositionsSupport.AsObject;
  static serializeBinaryToWriter(message: PositionsSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PositionsSupport;
  static deserializeBinaryFromReader(message: PositionsSupport, reader: jspb.BinaryReader): PositionsSupport;
}

export namespace PositionsSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    openPercentAttributes?: types_number_pb.FloatAttributes.AsObject;
    directionsList: Array<OpenClosePosition.Direction>;
    supportsStop: boolean;
    presetsList: Array<OpenClosePositions.Preset.AsObject>;
  };
}

export class GetOpenClosePositionsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetOpenClosePositionsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetOpenClosePositionsRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetOpenClosePositionsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetOpenClosePositionsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetOpenClosePositionsRequest): GetOpenClosePositionsRequest.AsObject;
  static serializeBinaryToWriter(message: GetOpenClosePositionsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetOpenClosePositionsRequest;
  static deserializeBinaryFromReader(message: GetOpenClosePositionsRequest, reader: jspb.BinaryReader): GetOpenClosePositionsRequest;
}

export namespace GetOpenClosePositionsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateOpenClosePositionsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateOpenClosePositionsRequest;

  getStates(): OpenClosePositions | undefined;
  setStates(value?: OpenClosePositions): UpdateOpenClosePositionsRequest;
  hasStates(): boolean;
  clearStates(): UpdateOpenClosePositionsRequest;

  getDelta(): boolean;
  setDelta(value: boolean): UpdateOpenClosePositionsRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateOpenClosePositionsRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateOpenClosePositionsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateOpenClosePositionsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateOpenClosePositionsRequest): UpdateOpenClosePositionsRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateOpenClosePositionsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateOpenClosePositionsRequest;
  static deserializeBinaryFromReader(message: UpdateOpenClosePositionsRequest, reader: jspb.BinaryReader): UpdateOpenClosePositionsRequest;
}

export namespace UpdateOpenClosePositionsRequest {
  export type AsObject = {
    name: string;
    states?: OpenClosePositions.AsObject;
    delta: boolean;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class StopOpenCloseRequest extends jspb.Message {
  getName(): string;
  setName(value: string): StopOpenCloseRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StopOpenCloseRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StopOpenCloseRequest): StopOpenCloseRequest.AsObject;
  static serializeBinaryToWriter(message: StopOpenCloseRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StopOpenCloseRequest;
  static deserializeBinaryFromReader(message: StopOpenCloseRequest, reader: jspb.BinaryReader): StopOpenCloseRequest;
}

export namespace StopOpenCloseRequest {
  export type AsObject = {
    name: string;
  };
}

export class PullOpenClosePositionsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullOpenClosePositionsRequest;

  getExcludeTweening(): boolean;
  setExcludeTweening(value: boolean): PullOpenClosePositionsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullOpenClosePositionsRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullOpenClosePositionsRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullOpenClosePositionsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullOpenClosePositionsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullOpenClosePositionsRequest): PullOpenClosePositionsRequest.AsObject;
  static serializeBinaryToWriter(message: PullOpenClosePositionsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullOpenClosePositionsRequest;
  static deserializeBinaryFromReader(message: PullOpenClosePositionsRequest, reader: jspb.BinaryReader): PullOpenClosePositionsRequest;
}

export namespace PullOpenClosePositionsRequest {
  export type AsObject = {
    name: string;
    excludeTweening: boolean;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullOpenClosePositionsResponse extends jspb.Message {
  getChangesList(): Array<PullOpenClosePositionsResponse.Change>;
  setChangesList(value: Array<PullOpenClosePositionsResponse.Change>): PullOpenClosePositionsResponse;
  clearChangesList(): PullOpenClosePositionsResponse;
  addChanges(value?: PullOpenClosePositionsResponse.Change, index?: number): PullOpenClosePositionsResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullOpenClosePositionsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullOpenClosePositionsResponse): PullOpenClosePositionsResponse.AsObject;
  static serializeBinaryToWriter(message: PullOpenClosePositionsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullOpenClosePositionsResponse;
  static deserializeBinaryFromReader(message: PullOpenClosePositionsResponse, reader: jspb.BinaryReader): PullOpenClosePositionsResponse;
}

export namespace PullOpenClosePositionsResponse {
  export type AsObject = {
    changesList: Array<PullOpenClosePositionsResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getOpenClosePosition(): OpenClosePositions | undefined;
    setOpenClosePosition(value?: OpenClosePositions): Change;
    hasOpenClosePosition(): boolean;
    clearOpenClosePosition(): Change;

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
      openClosePosition?: OpenClosePositions.AsObject;
    };
  }

}

export class DescribePositionsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribePositionsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribePositionsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribePositionsRequest): DescribePositionsRequest.AsObject;
  static serializeBinaryToWriter(message: DescribePositionsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribePositionsRequest;
  static deserializeBinaryFromReader(message: DescribePositionsRequest, reader: jspb.BinaryReader): DescribePositionsRequest;
}

export namespace DescribePositionsRequest {
  export type AsObject = {
    name: string;
  };
}

