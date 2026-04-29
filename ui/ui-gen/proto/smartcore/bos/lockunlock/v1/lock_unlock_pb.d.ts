import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class LockUnlock extends jspb.Message {
  getPosition(): LockUnlock.Position;
  setPosition(value: LockUnlock.Position): LockUnlock;

  getJammed(): boolean;
  setJammed(value: boolean): LockUnlock;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LockUnlock.AsObject;
  static toObject(includeInstance: boolean, msg: LockUnlock): LockUnlock.AsObject;
  static serializeBinaryToWriter(message: LockUnlock, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LockUnlock;
  static deserializeBinaryFromReader(message: LockUnlock, reader: jspb.BinaryReader): LockUnlock;
}

export namespace LockUnlock {
  export type AsObject = {
    position: LockUnlock.Position;
    jammed: boolean;
  };

  export enum Position {
    POSITION_UNSPECIFIED = 0,
    LOCKED = 1,
    UNLOCKED = 2,
    LOCKING = 3,
    UNLOCKING = 4,
  }
}

export class GetLockUnlockRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetLockUnlockRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetLockUnlockRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetLockUnlockRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetLockUnlockRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetLockUnlockRequest): GetLockUnlockRequest.AsObject;
  static serializeBinaryToWriter(message: GetLockUnlockRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetLockUnlockRequest;
  static deserializeBinaryFromReader(message: GetLockUnlockRequest, reader: jspb.BinaryReader): GetLockUnlockRequest;
}

export namespace GetLockUnlockRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateLockUnlockRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateLockUnlockRequest;

  getLockUnlock(): LockUnlock | undefined;
  setLockUnlock(value?: LockUnlock): UpdateLockUnlockRequest;
  hasLockUnlock(): boolean;
  clearLockUnlock(): UpdateLockUnlockRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateLockUnlockRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateLockUnlockRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateLockUnlockRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateLockUnlockRequest): UpdateLockUnlockRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateLockUnlockRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateLockUnlockRequest;
  static deserializeBinaryFromReader(message: UpdateLockUnlockRequest, reader: jspb.BinaryReader): UpdateLockUnlockRequest;
}

export namespace UpdateLockUnlockRequest {
  export type AsObject = {
    name: string;
    lockUnlock?: LockUnlock.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullLockUnlockRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullLockUnlockRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullLockUnlockRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullLockUnlockRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullLockUnlockRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullLockUnlockRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullLockUnlockRequest): PullLockUnlockRequest.AsObject;
  static serializeBinaryToWriter(message: PullLockUnlockRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullLockUnlockRequest;
  static deserializeBinaryFromReader(message: PullLockUnlockRequest, reader: jspb.BinaryReader): PullLockUnlockRequest;
}

export namespace PullLockUnlockRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullLockUnlockResponse extends jspb.Message {
  getChangesList(): Array<PullLockUnlockResponse.Change>;
  setChangesList(value: Array<PullLockUnlockResponse.Change>): PullLockUnlockResponse;
  clearChangesList(): PullLockUnlockResponse;
  addChanges(value?: PullLockUnlockResponse.Change, index?: number): PullLockUnlockResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullLockUnlockResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullLockUnlockResponse): PullLockUnlockResponse.AsObject;
  static serializeBinaryToWriter(message: PullLockUnlockResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullLockUnlockResponse;
  static deserializeBinaryFromReader(message: PullLockUnlockResponse, reader: jspb.BinaryReader): PullLockUnlockResponse;
}

export namespace PullLockUnlockResponse {
  export type AsObject = {
    changesList: Array<PullLockUnlockResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getLockUnlock(): LockUnlock | undefined;
    setLockUnlock(value?: LockUnlock): Change;
    hasLockUnlock(): boolean;
    clearLockUnlock(): Change;

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
      lockUnlock?: LockUnlock.AsObject;
    };
  }

}

