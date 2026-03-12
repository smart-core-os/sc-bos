import * as jspb from 'google-protobuf'

import * as google_protobuf_duration_pb from 'google-protobuf/google/protobuf/duration_pb'; // proto import: "google/protobuf/duration.proto"
import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_actor_v1_actor_pb from '../../../../smartcore/bos/actor/v1/actor_pb'; // proto import: "smartcore/bos/actor/v1/actor.proto"


export class RebootState extends jspb.Message {
  getBootTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setBootTime(value?: google_protobuf_timestamp_pb.Timestamp): RebootState;
  hasBootTime(): boolean;
  clearBootTime(): RebootState;

  getLastRebootReason(): string;
  setLastRebootReason(value: string): RebootState;

  getLastRebootActor(): string;
  setLastRebootActor(value: string): RebootState;

  getUptime(): google_protobuf_duration_pb.Duration | undefined;
  setUptime(value?: google_protobuf_duration_pb.Duration): RebootState;
  hasUptime(): boolean;
  clearUptime(): RebootState;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RebootState.AsObject;
  static toObject(includeInstance: boolean, msg: RebootState): RebootState.AsObject;
  static serializeBinaryToWriter(message: RebootState, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RebootState;
  static deserializeBinaryFromReader(message: RebootState, reader: jspb.BinaryReader): RebootState;
}

export namespace RebootState {
  export type AsObject = {
    bootTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    lastRebootReason: string;
    lastRebootActor: string;
    uptime?: google_protobuf_duration_pb.Duration.AsObject;
  };
}

export class GetRebootStateRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetRebootStateRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetRebootStateRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetRebootStateRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetRebootStateRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetRebootStateRequest): GetRebootStateRequest.AsObject;
  static serializeBinaryToWriter(message: GetRebootStateRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetRebootStateRequest;
  static deserializeBinaryFromReader(message: GetRebootStateRequest, reader: jspb.BinaryReader): GetRebootStateRequest;
}

export namespace GetRebootStateRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullRebootStateRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullRebootStateRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullRebootStateRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullRebootStateRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullRebootStateRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullRebootStateRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullRebootStateRequest): PullRebootStateRequest.AsObject;
  static serializeBinaryToWriter(message: PullRebootStateRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullRebootStateRequest;
  static deserializeBinaryFromReader(message: PullRebootStateRequest, reader: jspb.BinaryReader): PullRebootStateRequest;
}

export namespace PullRebootStateRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullRebootStateResponse extends jspb.Message {
  getChangesList(): Array<PullRebootStateResponse.Change>;
  setChangesList(value: Array<PullRebootStateResponse.Change>): PullRebootStateResponse;
  clearChangesList(): PullRebootStateResponse;
  addChanges(value?: PullRebootStateResponse.Change, index?: number): PullRebootStateResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullRebootStateResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullRebootStateResponse): PullRebootStateResponse.AsObject;
  static serializeBinaryToWriter(message: PullRebootStateResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullRebootStateResponse;
  static deserializeBinaryFromReader(message: PullRebootStateResponse, reader: jspb.BinaryReader): PullRebootStateResponse;
}

export namespace PullRebootStateResponse {
  export type AsObject = {
    changesList: Array<PullRebootStateResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getRebootState(): RebootState | undefined;
    setRebootState(value?: RebootState): Change;
    hasRebootState(): boolean;
    clearRebootState(): Change;

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
      rebootState?: RebootState.AsObject;
    };
  }

}

export class RebootRequest extends jspb.Message {
  getName(): string;
  setName(value: string): RebootRequest;

  getReason(): string;
  setReason(value: string): RebootRequest;

  getActor(): smartcore_bos_actor_v1_actor_pb.Actor | undefined;
  setActor(value?: smartcore_bos_actor_v1_actor_pb.Actor): RebootRequest;
  hasActor(): boolean;
  clearActor(): RebootRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RebootRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RebootRequest): RebootRequest.AsObject;
  static serializeBinaryToWriter(message: RebootRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RebootRequest;
  static deserializeBinaryFromReader(message: RebootRequest, reader: jspb.BinaryReader): RebootRequest;
}

export namespace RebootRequest {
  export type AsObject = {
    name: string;
    reason: string;
    actor?: smartcore_bos_actor_v1_actor_pb.Actor.AsObject;
  };
}

export class RebootResponse extends jspb.Message {
  getRebootTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRebootTime(value?: google_protobuf_timestamp_pb.Timestamp): RebootResponse;
  hasRebootTime(): boolean;
  clearRebootTime(): RebootResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RebootResponse.AsObject;
  static toObject(includeInstance: boolean, msg: RebootResponse): RebootResponse.AsObject;
  static serializeBinaryToWriter(message: RebootResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RebootResponse;
  static deserializeBinaryFromReader(message: RebootResponse, reader: jspb.BinaryReader): RebootResponse;
}

export namespace RebootResponse {
  export type AsObject = {
    rebootTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

