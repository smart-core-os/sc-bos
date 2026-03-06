import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_change_pb from '../types/change_pb'; // proto import: "types/change.proto"


export class Child extends jspb.Message {
  getName(): string;
  setName(value: string): Child;

  getTraitsList(): Array<Trait>;
  setTraitsList(value: Array<Trait>): Child;
  clearTraitsList(): Child;
  addTraits(value?: Trait, index?: number): Trait;

  getParent(): string;
  setParent(value: string): Child;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Child.AsObject;
  static toObject(includeInstance: boolean, msg: Child): Child.AsObject;
  static serializeBinaryToWriter(message: Child, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Child;
  static deserializeBinaryFromReader(message: Child, reader: jspb.BinaryReader): Child;
}

export namespace Child {
  export type AsObject = {
    name: string;
    traitsList: Array<Trait.AsObject>;
    parent: string;
  };
}

export class Trait extends jspb.Message {
  getName(): string;
  setName(value: string): Trait;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Trait.AsObject;
  static toObject(includeInstance: boolean, msg: Trait): Trait.AsObject;
  static serializeBinaryToWriter(message: Trait, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Trait;
  static deserializeBinaryFromReader(message: Trait, reader: jspb.BinaryReader): Trait;
}

export namespace Trait {
  export type AsObject = {
    name: string;
  };
}

export class ListChildrenRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListChildrenRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListChildrenRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListChildrenRequest;

  getPageSize(): number;
  setPageSize(value: number): ListChildrenRequest;

  getPageToken(): string;
  setPageToken(value: string): ListChildrenRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListChildrenRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListChildrenRequest): ListChildrenRequest.AsObject;
  static serializeBinaryToWriter(message: ListChildrenRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListChildrenRequest;
  static deserializeBinaryFromReader(message: ListChildrenRequest, reader: jspb.BinaryReader): ListChildrenRequest;
}

export namespace ListChildrenRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
  };
}

export class ListChildrenResponse extends jspb.Message {
  getChildrenList(): Array<Child>;
  setChildrenList(value: Array<Child>): ListChildrenResponse;
  clearChildrenList(): ListChildrenResponse;
  addChildren(value?: Child, index?: number): Child;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListChildrenResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListChildrenResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListChildrenResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListChildrenResponse): ListChildrenResponse.AsObject;
  static serializeBinaryToWriter(message: ListChildrenResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListChildrenResponse;
  static deserializeBinaryFromReader(message: ListChildrenResponse, reader: jspb.BinaryReader): ListChildrenResponse;
}

export namespace ListChildrenResponse {
  export type AsObject = {
    childrenList: Array<Child.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class PullChildrenRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullChildrenRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullChildrenRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullChildrenRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullChildrenRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullChildrenRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullChildrenRequest): PullChildrenRequest.AsObject;
  static serializeBinaryToWriter(message: PullChildrenRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullChildrenRequest;
  static deserializeBinaryFromReader(message: PullChildrenRequest, reader: jspb.BinaryReader): PullChildrenRequest;
}

export namespace PullChildrenRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullChildrenResponse extends jspb.Message {
  getChangesList(): Array<PullChildrenResponse.Change>;
  setChangesList(value: Array<PullChildrenResponse.Change>): PullChildrenResponse;
  clearChangesList(): PullChildrenResponse;
  addChanges(value?: PullChildrenResponse.Change, index?: number): PullChildrenResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullChildrenResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullChildrenResponse): PullChildrenResponse.AsObject;
  static serializeBinaryToWriter(message: PullChildrenResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullChildrenResponse;
  static deserializeBinaryFromReader(message: PullChildrenResponse, reader: jspb.BinaryReader): PullChildrenResponse;
}

export namespace PullChildrenResponse {
  export type AsObject = {
    changesList: Array<PullChildrenResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getType(): types_change_pb.ChangeType;
    setType(value: types_change_pb.ChangeType): Change;

    getNewValue(): Child | undefined;
    setNewValue(value?: Child): Change;
    hasNewValue(): boolean;
    clearNewValue(): Change;

    getOldValue(): Child | undefined;
    setOldValue(value?: Child): Change;
    hasOldValue(): boolean;
    clearOldValue(): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

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
      type: types_change_pb.ChangeType;
      newValue?: Child.AsObject;
      oldValue?: Child.AsObject;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };
  }

}

