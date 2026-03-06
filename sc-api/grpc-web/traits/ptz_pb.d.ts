import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"
import * as types_tween_pb from '../types/tween_pb'; // proto import: "types/tween.proto"


export class Ptz extends jspb.Message {
  getPosition(): PtzPosition | undefined;
  setPosition(value?: PtzPosition): Ptz;
  hasPosition(): boolean;
  clearPosition(): Ptz;

  getMovement(): PtzMovement | undefined;
  setMovement(value?: PtzMovement): Ptz;
  hasMovement(): boolean;
  clearMovement(): Ptz;

  getPreset(): string;
  setPreset(value: string): Ptz;

  getPresetSpeed(): number;
  setPresetSpeed(value: number): Ptz;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Ptz.AsObject;
  static toObject(includeInstance: boolean, msg: Ptz): Ptz.AsObject;
  static serializeBinaryToWriter(message: Ptz, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Ptz;
  static deserializeBinaryFromReader(message: Ptz, reader: jspb.BinaryReader): Ptz;
}

export namespace Ptz {
  export type AsObject = {
    position?: PtzPosition.AsObject;
    movement?: PtzMovement.AsObject;
    preset: string;
    presetSpeed: number;
  };
}

export class PtzVector extends jspb.Message {
  getPan(): number;
  setPan(value: number): PtzVector;

  getTilt(): number;
  setTilt(value: number): PtzVector;

  getZoom(): number;
  setZoom(value: number): PtzVector;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PtzVector.AsObject;
  static toObject(includeInstance: boolean, msg: PtzVector): PtzVector.AsObject;
  static serializeBinaryToWriter(message: PtzVector, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PtzVector;
  static deserializeBinaryFromReader(message: PtzVector, reader: jspb.BinaryReader): PtzVector;
}

export namespace PtzVector {
  export type AsObject = {
    pan: number;
    tilt: number;
    zoom: number;
  };
}

export class PtzBounds extends jspb.Message {
  getMin(): PtzVector | undefined;
  setMin(value?: PtzVector): PtzBounds;
  hasMin(): boolean;
  clearMin(): PtzBounds;

  getMax(): PtzVector | undefined;
  setMax(value?: PtzVector): PtzBounds;
  hasMax(): boolean;
  clearMax(): PtzBounds;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PtzBounds.AsObject;
  static toObject(includeInstance: boolean, msg: PtzBounds): PtzBounds.AsObject;
  static serializeBinaryToWriter(message: PtzBounds, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PtzBounds;
  static deserializeBinaryFromReader(message: PtzBounds, reader: jspb.BinaryReader): PtzBounds;
}

export namespace PtzBounds {
  export type AsObject = {
    min?: PtzVector.AsObject;
    max?: PtzVector.AsObject;
  };
}

export class PtzMovement extends jspb.Message {
  getDirection(): PtzVector | undefined;
  setDirection(value?: PtzVector): PtzMovement;
  hasDirection(): boolean;
  clearDirection(): PtzMovement;

  getSpeed(): number;
  setSpeed(value: number): PtzMovement;

  getSpeedTween(): types_tween_pb.Tween | undefined;
  setSpeedTween(value?: types_tween_pb.Tween): PtzMovement;
  hasSpeedTween(): boolean;
  clearSpeedTween(): PtzMovement;

  getTargetSpeed(): number;
  setTargetSpeed(value: number): PtzMovement;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PtzMovement.AsObject;
  static toObject(includeInstance: boolean, msg: PtzMovement): PtzMovement.AsObject;
  static serializeBinaryToWriter(message: PtzMovement, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PtzMovement;
  static deserializeBinaryFromReader(message: PtzMovement, reader: jspb.BinaryReader): PtzMovement;
}

export namespace PtzMovement {
  export type AsObject = {
    direction?: PtzVector.AsObject;
    speed: number;
    speedTween?: types_tween_pb.Tween.AsObject;
    targetSpeed: number;
  };
}

export class PtzPosition extends jspb.Message {
  getPosition(): PtzVector | undefined;
  setPosition(value?: PtzVector): PtzPosition;
  hasPosition(): boolean;
  clearPosition(): PtzPosition;

  getTween(): types_tween_pb.Tween | undefined;
  setTween(value?: types_tween_pb.Tween): PtzPosition;
  hasTween(): boolean;
  clearTween(): PtzPosition;

  getTargetPosition(): PtzVector | undefined;
  setTargetPosition(value?: PtzVector): PtzPosition;
  hasTargetPosition(): boolean;
  clearTargetPosition(): PtzPosition;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PtzPosition.AsObject;
  static toObject(includeInstance: boolean, msg: PtzPosition): PtzPosition.AsObject;
  static serializeBinaryToWriter(message: PtzPosition, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PtzPosition;
  static deserializeBinaryFromReader(message: PtzPosition, reader: jspb.BinaryReader): PtzPosition;
}

export namespace PtzPosition {
  export type AsObject = {
    position?: PtzVector.AsObject;
    tween?: types_tween_pb.Tween.AsObject;
    targetPosition?: PtzVector.AsObject;
  };
}

export class PtzPreset extends jspb.Message {
  getName(): string;
  setName(value: string): PtzPreset;

  getTitle(): string;
  setTitle(value: string): PtzPreset;

  getDescription(): string;
  setDescription(value: string): PtzPreset;

  getPosition(): PtzVector | undefined;
  setPosition(value?: PtzVector): PtzPreset;
  hasPosition(): boolean;
  clearPosition(): PtzPreset;

  getWritable(): boolean;
  setWritable(value: boolean): PtzPreset;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PtzPreset.AsObject;
  static toObject(includeInstance: boolean, msg: PtzPreset): PtzPreset.AsObject;
  static serializeBinaryToWriter(message: PtzPreset, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PtzPreset;
  static deserializeBinaryFromReader(message: PtzPreset, reader: jspb.BinaryReader): PtzPreset;
}

export namespace PtzPreset {
  export type AsObject = {
    name: string;
    title: string;
    description: string;
    position?: PtzVector.AsObject;
    writable: boolean;
  };
}

export class PtzSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): PtzSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): PtzSupport;

  getSupportsPosition(): boolean;
  setSupportsPosition(value: boolean): PtzSupport;

  getSupportsMovement(): boolean;
  setSupportsMovement(value: boolean): PtzSupport;

  getPresetsList(): Array<PtzPreset>;
  setPresetsList(value: Array<PtzPreset>): PtzSupport;
  clearPresetsList(): PtzSupport;
  addPresets(value?: PtzPreset, index?: number): PtzPreset;

  getSupportsCustomPresets(): boolean;
  setSupportsCustomPresets(value: boolean): PtzSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PtzSupport.AsObject;
  static toObject(includeInstance: boolean, msg: PtzSupport): PtzSupport.AsObject;
  static serializeBinaryToWriter(message: PtzSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PtzSupport;
  static deserializeBinaryFromReader(message: PtzSupport, reader: jspb.BinaryReader): PtzSupport;
}

export namespace PtzSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    supportsPosition: boolean;
    supportsMovement: boolean;
    presetsList: Array<PtzPreset.AsObject>;
    supportsCustomPresets: boolean;
  };
}

export class GetPtzRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetPtzRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetPtzRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetPtzRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetPtzRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetPtzRequest): GetPtzRequest.AsObject;
  static serializeBinaryToWriter(message: GetPtzRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetPtzRequest;
  static deserializeBinaryFromReader(message: GetPtzRequest, reader: jspb.BinaryReader): GetPtzRequest;
}

export namespace GetPtzRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdatePtzRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdatePtzRequest;

  getState(): Ptz | undefined;
  setState(value?: Ptz): UpdatePtzRequest;
  hasState(): boolean;
  clearState(): UpdatePtzRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdatePtzRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdatePtzRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdatePtzRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdatePtzRequest): UpdatePtzRequest.AsObject;
  static serializeBinaryToWriter(message: UpdatePtzRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdatePtzRequest;
  static deserializeBinaryFromReader(message: UpdatePtzRequest, reader: jspb.BinaryReader): UpdatePtzRequest;
}

export namespace UpdatePtzRequest {
  export type AsObject = {
    name: string;
    state?: Ptz.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class StopPtzRequest extends jspb.Message {
  getName(): string;
  setName(value: string): StopPtzRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StopPtzRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StopPtzRequest): StopPtzRequest.AsObject;
  static serializeBinaryToWriter(message: StopPtzRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StopPtzRequest;
  static deserializeBinaryFromReader(message: StopPtzRequest, reader: jspb.BinaryReader): StopPtzRequest;
}

export namespace StopPtzRequest {
  export type AsObject = {
    name: string;
  };
}

export class CreatePtzPresetRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CreatePtzPresetRequest;

  getPreset(): PtzPreset | undefined;
  setPreset(value?: PtzPreset): CreatePtzPresetRequest;
  hasPreset(): boolean;
  clearPreset(): CreatePtzPresetRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreatePtzPresetRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreatePtzPresetRequest): CreatePtzPresetRequest.AsObject;
  static serializeBinaryToWriter(message: CreatePtzPresetRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreatePtzPresetRequest;
  static deserializeBinaryFromReader(message: CreatePtzPresetRequest, reader: jspb.BinaryReader): CreatePtzPresetRequest;
}

export namespace CreatePtzPresetRequest {
  export type AsObject = {
    name: string;
    preset?: PtzPreset.AsObject;
  };
}

export class PullPtzRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullPtzRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullPtzRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullPtzRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullPtzRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullPtzRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullPtzRequest): PullPtzRequest.AsObject;
  static serializeBinaryToWriter(message: PullPtzRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullPtzRequest;
  static deserializeBinaryFromReader(message: PullPtzRequest, reader: jspb.BinaryReader): PullPtzRequest;
}

export namespace PullPtzRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullPtzResponse extends jspb.Message {
  getChangesList(): Array<PullPtzResponse.Change>;
  setChangesList(value: Array<PullPtzResponse.Change>): PullPtzResponse;
  clearChangesList(): PullPtzResponse;
  addChanges(value?: PullPtzResponse.Change, index?: number): PullPtzResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullPtzResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullPtzResponse): PullPtzResponse.AsObject;
  static serializeBinaryToWriter(message: PullPtzResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullPtzResponse;
  static deserializeBinaryFromReader(message: PullPtzResponse, reader: jspb.BinaryReader): PullPtzResponse;
}

export namespace PullPtzResponse {
  export type AsObject = {
    changesList: Array<PullPtzResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getPtz(): Ptz | undefined;
    setPtz(value?: Ptz): Change;
    hasPtz(): boolean;
    clearPtz(): Change;

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
      ptz?: Ptz.AsObject;
    };
  }

}

export class DescribePtzRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribePtzRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribePtzRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribePtzRequest): DescribePtzRequest.AsObject;
  static serializeBinaryToWriter(message: DescribePtzRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribePtzRequest;
  static deserializeBinaryFromReader(message: DescribePtzRequest, reader: jspb.BinaryReader): DescribePtzRequest;
}

export namespace DescribePtzRequest {
  export type AsObject = {
    name: string;
  };
}

