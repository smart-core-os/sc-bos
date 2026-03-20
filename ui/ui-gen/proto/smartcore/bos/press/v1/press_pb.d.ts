import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class PressedState extends jspb.Message {
  getState(): PressedState.Press;
  setState(value: PressedState.Press): PressedState;

  getStateChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStateChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): PressedState;
  hasStateChangeTime(): boolean;
  clearStateChangeTime(): PressedState;

  getMostRecentGesture(): PressedState.Gesture | undefined;
  setMostRecentGesture(value?: PressedState.Gesture): PressedState;
  hasMostRecentGesture(): boolean;
  clearMostRecentGesture(): PressedState;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PressedState.AsObject;
  static toObject(includeInstance: boolean, msg: PressedState): PressedState.AsObject;
  static serializeBinaryToWriter(message: PressedState, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PressedState;
  static deserializeBinaryFromReader(message: PressedState, reader: jspb.BinaryReader): PressedState;
}

export namespace PressedState {
  export type AsObject = {
    state: PressedState.Press;
    stateChangeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    mostRecentGesture?: PressedState.Gesture.AsObject;
  };

  export class Gesture extends jspb.Message {
    getId(): string;
    setId(value: string): Gesture;

    getKind(): PressedState.Gesture.Kind;
    setKind(value: PressedState.Gesture.Kind): Gesture;

    getCount(): number;
    setCount(value: number): Gesture;

    getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): Gesture;
    hasStartTime(): boolean;
    clearStartTime(): Gesture;

    getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): Gesture;
    hasEndTime(): boolean;
    clearEndTime(): Gesture;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Gesture.AsObject;
    static toObject(includeInstance: boolean, msg: Gesture): Gesture.AsObject;
    static serializeBinaryToWriter(message: Gesture, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Gesture;
    static deserializeBinaryFromReader(message: Gesture, reader: jspb.BinaryReader): Gesture;
  }

  export namespace Gesture {
    export type AsObject = {
      id: string;
      kind: PressedState.Gesture.Kind;
      count: number;
      startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };

    export enum Kind {
      KIND_UNSPECIFIED = 0,
      CLICK = 1,
      HOLD = 2,
    }
  }


  export enum Press {
    PRESS_UNSPECIFIED = 0,
    UNPRESSED = 1,
    PRESSED = 2,
  }
}

export class GetPressedStateRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetPressedStateRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetPressedStateRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetPressedStateRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetPressedStateRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetPressedStateRequest): GetPressedStateRequest.AsObject;
  static serializeBinaryToWriter(message: GetPressedStateRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetPressedStateRequest;
  static deserializeBinaryFromReader(message: GetPressedStateRequest, reader: jspb.BinaryReader): GetPressedStateRequest;
}

export namespace GetPressedStateRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullPressedStateRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullPressedStateRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullPressedStateRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullPressedStateRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullPressedStateRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullPressedStateRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullPressedStateRequest): PullPressedStateRequest.AsObject;
  static serializeBinaryToWriter(message: PullPressedStateRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullPressedStateRequest;
  static deserializeBinaryFromReader(message: PullPressedStateRequest, reader: jspb.BinaryReader): PullPressedStateRequest;
}

export namespace PullPressedStateRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullPressedStateResponse extends jspb.Message {
  getChangesList(): Array<PullPressedStateResponse.Change>;
  setChangesList(value: Array<PullPressedStateResponse.Change>): PullPressedStateResponse;
  clearChangesList(): PullPressedStateResponse;
  addChanges(value?: PullPressedStateResponse.Change, index?: number): PullPressedStateResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullPressedStateResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullPressedStateResponse): PullPressedStateResponse.AsObject;
  static serializeBinaryToWriter(message: PullPressedStateResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullPressedStateResponse;
  static deserializeBinaryFromReader(message: PullPressedStateResponse, reader: jspb.BinaryReader): PullPressedStateResponse;
}

export namespace PullPressedStateResponse {
  export type AsObject = {
    changesList: Array<PullPressedStateResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getPressedState(): PressedState | undefined;
    setPressedState(value?: PressedState): Change;
    hasPressedState(): boolean;
    clearPressedState(): Change;

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
      pressedState?: PressedState.AsObject;
    };
  }

}

export class UpdatePressedStateRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdatePressedStateRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdatePressedStateRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdatePressedStateRequest;

  getPressedState(): PressedState | undefined;
  setPressedState(value?: PressedState): UpdatePressedStateRequest;
  hasPressedState(): boolean;
  clearPressedState(): UpdatePressedStateRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdatePressedStateRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdatePressedStateRequest): UpdatePressedStateRequest.AsObject;
  static serializeBinaryToWriter(message: UpdatePressedStateRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdatePressedStateRequest;
  static deserializeBinaryFromReader(message: UpdatePressedStateRequest, reader: jspb.BinaryReader): UpdatePressedStateRequest;
}

export namespace UpdatePressedStateRequest {
  export type AsObject = {
    name: string;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pressedState?: PressedState.AsObject;
  };
}

