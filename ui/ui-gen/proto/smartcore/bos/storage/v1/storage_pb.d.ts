import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class Storage extends jspb.Message {
  getBytes(): StorageBytes | undefined;
  setBytes(value?: StorageBytes): Storage;
  hasBytes(): boolean;
  clearBytes(): Storage;

  getItems(): StorageItems | undefined;
  setItems(value?: StorageItems): Storage;
  hasItems(): boolean;
  clearItems(): Storage;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Storage.AsObject;
  static toObject(includeInstance: boolean, msg: Storage): Storage.AsObject;
  static serializeBinaryToWriter(message: Storage, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Storage;
  static deserializeBinaryFromReader(message: Storage, reader: jspb.BinaryReader): Storage;
}

export namespace Storage {
  export type AsObject = {
    bytes?: StorageBytes.AsObject;
    items?: StorageItems.AsObject;
  };
}

export class StorageBytes extends jspb.Message {
  getUsed(): number;
  setUsed(value: number): StorageBytes;
  hasUsed(): boolean;
  clearUsed(): StorageBytes;

  getCapacity(): number;
  setCapacity(value: number): StorageBytes;
  hasCapacity(): boolean;
  clearCapacity(): StorageBytes;

  getUtilization(): number;
  setUtilization(value: number): StorageBytes;
  hasUtilization(): boolean;
  clearUtilization(): StorageBytes;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StorageBytes.AsObject;
  static toObject(includeInstance: boolean, msg: StorageBytes): StorageBytes.AsObject;
  static serializeBinaryToWriter(message: StorageBytes, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StorageBytes;
  static deserializeBinaryFromReader(message: StorageBytes, reader: jspb.BinaryReader): StorageBytes;
}

export namespace StorageBytes {
  export type AsObject = {
    used?: number;
    capacity?: number;
    utilization?: number;
  };

  export enum UsedCase {
    _USED_NOT_SET = 0,
    USED = 1,
  }

  export enum CapacityCase {
    _CAPACITY_NOT_SET = 0,
    CAPACITY = 2,
  }

  export enum UtilizationCase {
    _UTILIZATION_NOT_SET = 0,
    UTILIZATION = 3,
  }
}

export class StorageItems extends jspb.Message {
  getUsed(): number;
  setUsed(value: number): StorageItems;
  hasUsed(): boolean;
  clearUsed(): StorageItems;

  getCapacity(): number;
  setCapacity(value: number): StorageItems;
  hasCapacity(): boolean;
  clearCapacity(): StorageItems;

  getUtilization(): number;
  setUtilization(value: number): StorageItems;
  hasUtilization(): boolean;
  clearUtilization(): StorageItems;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StorageItems.AsObject;
  static toObject(includeInstance: boolean, msg: StorageItems): StorageItems.AsObject;
  static serializeBinaryToWriter(message: StorageItems, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StorageItems;
  static deserializeBinaryFromReader(message: StorageItems, reader: jspb.BinaryReader): StorageItems;
}

export namespace StorageItems {
  export type AsObject = {
    used?: number;
    capacity?: number;
    utilization?: number;
  };

  export enum UsedCase {
    _USED_NOT_SET = 0,
    USED = 1,
  }

  export enum CapacityCase {
    _CAPACITY_NOT_SET = 0,
    CAPACITY = 2,
  }

  export enum UtilizationCase {
    _UTILIZATION_NOT_SET = 0,
    UTILIZATION = 3,
  }
}

export class StorageSupport extends jspb.Message {
  getSupportedActionsList(): Array<StorageAdminAction>;
  setSupportedActionsList(value: Array<StorageAdminAction>): StorageSupport;
  clearSupportedActionsList(): StorageSupport;
  addSupportedActions(value: StorageAdminAction, index?: number): StorageSupport;

  getItemName(): string;
  setItemName(value: string): StorageSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StorageSupport.AsObject;
  static toObject(includeInstance: boolean, msg: StorageSupport): StorageSupport.AsObject;
  static serializeBinaryToWriter(message: StorageSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StorageSupport;
  static deserializeBinaryFromReader(message: StorageSupport, reader: jspb.BinaryReader): StorageSupport;
}

export namespace StorageSupport {
  export type AsObject = {
    supportedActionsList: Array<StorageAdminAction>;
    itemName: string;
  };
}

export class GetStorageRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetStorageRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetStorageRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetStorageRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetStorageRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetStorageRequest): GetStorageRequest.AsObject;
  static serializeBinaryToWriter(message: GetStorageRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetStorageRequest;
  static deserializeBinaryFromReader(message: GetStorageRequest, reader: jspb.BinaryReader): GetStorageRequest;
}

export namespace GetStorageRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullStorageRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullStorageRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullStorageRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullStorageRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullStorageRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullStorageRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullStorageRequest): PullStorageRequest.AsObject;
  static serializeBinaryToWriter(message: PullStorageRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullStorageRequest;
  static deserializeBinaryFromReader(message: PullStorageRequest, reader: jspb.BinaryReader): PullStorageRequest;
}

export namespace PullStorageRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullStorageResponse extends jspb.Message {
  getChangesList(): Array<PullStorageResponse.Change>;
  setChangesList(value: Array<PullStorageResponse.Change>): PullStorageResponse;
  clearChangesList(): PullStorageResponse;
  addChanges(value?: PullStorageResponse.Change, index?: number): PullStorageResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullStorageResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullStorageResponse): PullStorageResponse.AsObject;
  static serializeBinaryToWriter(message: PullStorageResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullStorageResponse;
  static deserializeBinaryFromReader(message: PullStorageResponse, reader: jspb.BinaryReader): PullStorageResponse;
}

export namespace PullStorageResponse {
  export type AsObject = {
    changesList: Array<PullStorageResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getStorage(): Storage | undefined;
    setStorage(value?: Storage): Change;
    hasStorage(): boolean;
    clearStorage(): Change;

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
      storage?: Storage.AsObject;
    };
  }

}

export class PerformStorageAdminRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PerformStorageAdminRequest;

  getAction(): StorageAdminAction;
  setAction(value: StorageAdminAction): PerformStorageAdminRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PerformStorageAdminRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PerformStorageAdminRequest): PerformStorageAdminRequest.AsObject;
  static serializeBinaryToWriter(message: PerformStorageAdminRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PerformStorageAdminRequest;
  static deserializeBinaryFromReader(message: PerformStorageAdminRequest, reader: jspb.BinaryReader): PerformStorageAdminRequest;
}

export namespace PerformStorageAdminRequest {
  export type AsObject = {
    name: string;
    action: StorageAdminAction;
  };
}

export class PerformStorageAdminResponse extends jspb.Message {
  getFreedBytes(): StorageBytes | undefined;
  setFreedBytes(value?: StorageBytes): PerformStorageAdminResponse;
  hasFreedBytes(): boolean;
  clearFreedBytes(): PerformStorageAdminResponse;

  getFreedItems(): StorageItems | undefined;
  setFreedItems(value?: StorageItems): PerformStorageAdminResponse;
  hasFreedItems(): boolean;
  clearFreedItems(): PerformStorageAdminResponse;

  getMessage(): string;
  setMessage(value: string): PerformStorageAdminResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PerformStorageAdminResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PerformStorageAdminResponse): PerformStorageAdminResponse.AsObject;
  static serializeBinaryToWriter(message: PerformStorageAdminResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PerformStorageAdminResponse;
  static deserializeBinaryFromReader(message: PerformStorageAdminResponse, reader: jspb.BinaryReader): PerformStorageAdminResponse;
}

export namespace PerformStorageAdminResponse {
  export type AsObject = {
    freedBytes?: StorageBytes.AsObject;
    freedItems?: StorageItems.AsObject;
    message: string;
  };
}

export class DescribeStorageRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeStorageRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeStorageRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeStorageRequest): DescribeStorageRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeStorageRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeStorageRequest;
  static deserializeBinaryFromReader(message: DescribeStorageRequest, reader: jspb.BinaryReader): DescribeStorageRequest;
}

export namespace DescribeStorageRequest {
  export type AsObject = {
    name: string;
  };
}

export enum StorageAdminAction {
  STORAGE_ADMIN_ACTION_UNSPECIFIED = 0,
  STORAGE_ADMIN_ACTION_CLEAR = 1,
  STORAGE_ADMIN_ACTION_DELETE_OLD = 2,
  STORAGE_ADMIN_ACTION_GC = 3,
  STORAGE_ADMIN_ACTION_COMPACT = 4,
  STORAGE_ADMIN_ACTION_VACUUM = 5,
}
