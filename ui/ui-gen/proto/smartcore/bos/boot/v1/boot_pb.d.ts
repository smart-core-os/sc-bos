import * as jspb from 'google-protobuf'

import * as google_protobuf_duration_pb from 'google-protobuf/google/protobuf/duration_pb'; // proto import: "google/protobuf/duration.proto"
import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_actor_v1_actor_pb from '../../../../smartcore/bos/actor/v1/actor_pb'; // proto import: "smartcore/bos/actor/v1/actor.proto"


export class BootState extends jspb.Message {
  getBootTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setBootTime(value?: google_protobuf_timestamp_pb.Timestamp): BootState;
  hasBootTime(): boolean;
  clearBootTime(): BootState;

  getLastRebootReason(): string;
  setLastRebootReason(value: string): BootState;

  getLastRebootActor(): string;
  setLastRebootActor(value: string): BootState;

  getUptime(): google_protobuf_duration_pb.Duration | undefined;
  setUptime(value?: google_protobuf_duration_pb.Duration): BootState;
  hasUptime(): boolean;
  clearUptime(): BootState;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BootState.AsObject;
  static toObject(includeInstance: boolean, msg: BootState): BootState.AsObject;
  static serializeBinaryToWriter(message: BootState, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BootState;
  static deserializeBinaryFromReader(message: BootState, reader: jspb.BinaryReader): BootState;
}

export namespace BootState {
  export type AsObject = {
    bootTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    lastRebootReason: string;
    lastRebootActor: string;
    uptime?: google_protobuf_duration_pb.Duration.AsObject;
  };
}

export class GetBootStateRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetBootStateRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetBootStateRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetBootStateRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBootStateRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetBootStateRequest): GetBootStateRequest.AsObject;
  static serializeBinaryToWriter(message: GetBootStateRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBootStateRequest;
  static deserializeBinaryFromReader(message: GetBootStateRequest, reader: jspb.BinaryReader): GetBootStateRequest;
}

export namespace GetBootStateRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullBootStateRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullBootStateRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullBootStateRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullBootStateRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullBootStateRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullBootStateRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullBootStateRequest): PullBootStateRequest.AsObject;
  static serializeBinaryToWriter(message: PullBootStateRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullBootStateRequest;
  static deserializeBinaryFromReader(message: PullBootStateRequest, reader: jspb.BinaryReader): PullBootStateRequest;
}

export namespace PullBootStateRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullBootStateResponse extends jspb.Message {
  getChangesList(): Array<PullBootStateResponse.Change>;
  setChangesList(value: Array<PullBootStateResponse.Change>): PullBootStateResponse;
  clearChangesList(): PullBootStateResponse;
  addChanges(value?: PullBootStateResponse.Change, index?: number): PullBootStateResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullBootStateResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullBootStateResponse): PullBootStateResponse.AsObject;
  static serializeBinaryToWriter(message: PullBootStateResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullBootStateResponse;
  static deserializeBinaryFromReader(message: PullBootStateResponse, reader: jspb.BinaryReader): PullBootStateResponse;
}

export namespace PullBootStateResponse {
  export type AsObject = {
    changesList: Array<PullBootStateResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getBootState(): BootState | undefined;
    setBootState(value?: BootState): Change;
    hasBootState(): boolean;
    clearBootState(): Change;

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
      bootState?: BootState.AsObject;
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

