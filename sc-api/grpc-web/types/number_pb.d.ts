import * as jspb from 'google-protobuf'

import * as types_tween_pb from '../types/tween_pb'; // proto import: "types/tween.proto"


export class NumberCapping extends jspb.Message {
  getMin(): InvalidNumberBehaviour;
  setMin(value: InvalidNumberBehaviour): NumberCapping;

  getStep(): InvalidNumberBehaviour;
  setStep(value: InvalidNumberBehaviour): NumberCapping;

  getMax(): InvalidNumberBehaviour;
  setMax(value: InvalidNumberBehaviour): NumberCapping;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NumberCapping.AsObject;
  static toObject(includeInstance: boolean, msg: NumberCapping): NumberCapping.AsObject;
  static serializeBinaryToWriter(message: NumberCapping, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NumberCapping;
  static deserializeBinaryFromReader(message: NumberCapping, reader: jspb.BinaryReader): NumberCapping;
}

export namespace NumberCapping {
  export type AsObject = {
    min: InvalidNumberBehaviour;
    step: InvalidNumberBehaviour;
    max: InvalidNumberBehaviour;
  };
}

export class Int32Bounds extends jspb.Message {
  getMin(): number;
  setMin(value: number): Int32Bounds;
  hasMin(): boolean;
  clearMin(): Int32Bounds;

  getMax(): number;
  setMax(value: number): Int32Bounds;
  hasMax(): boolean;
  clearMax(): Int32Bounds;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Int32Bounds.AsObject;
  static toObject(includeInstance: boolean, msg: Int32Bounds): Int32Bounds.AsObject;
  static serializeBinaryToWriter(message: Int32Bounds, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Int32Bounds;
  static deserializeBinaryFromReader(message: Int32Bounds, reader: jspb.BinaryReader): Int32Bounds;
}

export namespace Int32Bounds {
  export type AsObject = {
    min?: number;
    max?: number;
  };

  export enum MinCase {
    _MIN_NOT_SET = 0,
    MIN = 1,
  }

  export enum MaxCase {
    _MAX_NOT_SET = 0,
    MAX = 2,
  }
}

export class Int32Attributes extends jspb.Message {
  getBounds(): Int32Bounds | undefined;
  setBounds(value?: Int32Bounds): Int32Attributes;
  hasBounds(): boolean;
  clearBounds(): Int32Attributes;

  getStep(): number;
  setStep(value: number): Int32Attributes;

  getSupportsDelta(): boolean;
  setSupportsDelta(value: boolean): Int32Attributes;

  getRampSupport(): types_tween_pb.TweenSupport;
  setRampSupport(value: types_tween_pb.TweenSupport): Int32Attributes;

  getDefaultCapping(): NumberCapping | undefined;
  setDefaultCapping(value?: NumberCapping): Int32Attributes;
  hasDefaultCapping(): boolean;
  clearDefaultCapping(): Int32Attributes;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Int32Attributes.AsObject;
  static toObject(includeInstance: boolean, msg: Int32Attributes): Int32Attributes.AsObject;
  static serializeBinaryToWriter(message: Int32Attributes, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Int32Attributes;
  static deserializeBinaryFromReader(message: Int32Attributes, reader: jspb.BinaryReader): Int32Attributes;
}

export namespace Int32Attributes {
  export type AsObject = {
    bounds?: Int32Bounds.AsObject;
    step: number;
    supportsDelta: boolean;
    rampSupport: types_tween_pb.TweenSupport;
    defaultCapping?: NumberCapping.AsObject;
  };
}

export class FloatBounds extends jspb.Message {
  getMin(): number;
  setMin(value: number): FloatBounds;
  hasMin(): boolean;
  clearMin(): FloatBounds;

  getMax(): number;
  setMax(value: number): FloatBounds;
  hasMax(): boolean;
  clearMax(): FloatBounds;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): FloatBounds.AsObject;
  static toObject(includeInstance: boolean, msg: FloatBounds): FloatBounds.AsObject;
  static serializeBinaryToWriter(message: FloatBounds, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): FloatBounds;
  static deserializeBinaryFromReader(message: FloatBounds, reader: jspb.BinaryReader): FloatBounds;
}

export namespace FloatBounds {
  export type AsObject = {
    min?: number;
    max?: number;
  };

  export enum MinCase {
    _MIN_NOT_SET = 0,
    MIN = 1,
  }

  export enum MaxCase {
    _MAX_NOT_SET = 0,
    MAX = 2,
  }
}

export class FloatAttributes extends jspb.Message {
  getBounds(): FloatBounds | undefined;
  setBounds(value?: FloatBounds): FloatAttributes;
  hasBounds(): boolean;
  clearBounds(): FloatAttributes;

  getStep(): number;
  setStep(value: number): FloatAttributes;

  getSupportsDelta(): boolean;
  setSupportsDelta(value: boolean): FloatAttributes;

  getRampSupport(): types_tween_pb.TweenSupport;
  setRampSupport(value: types_tween_pb.TweenSupport): FloatAttributes;

  getDefaultCapping(): NumberCapping | undefined;
  setDefaultCapping(value?: NumberCapping): FloatAttributes;
  hasDefaultCapping(): boolean;
  clearDefaultCapping(): FloatAttributes;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): FloatAttributes.AsObject;
  static toObject(includeInstance: boolean, msg: FloatAttributes): FloatAttributes.AsObject;
  static serializeBinaryToWriter(message: FloatAttributes, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): FloatAttributes;
  static deserializeBinaryFromReader(message: FloatAttributes, reader: jspb.BinaryReader): FloatAttributes;
}

export namespace FloatAttributes {
  export type AsObject = {
    bounds?: FloatBounds.AsObject;
    step: number;
    supportsDelta: boolean;
    rampSupport: types_tween_pb.TweenSupport;
    defaultCapping?: NumberCapping.AsObject;
  };
}

export enum InvalidNumberBehaviour {
  INVALID_NUMBER_BEHAVIOUR_UNSPECIFIED = 0,
  RESTRICT = 1,
  ERROR = 2,
  ALLOW = 3,
}
