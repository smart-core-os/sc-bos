import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class DataRetention extends jspb.Message {
  getBytes(): DataRetentionBytes | undefined;
  setBytes(value?: DataRetentionBytes): DataRetention;
  hasBytes(): boolean;
  clearBytes(): DataRetention;

  getItems(): DataRetentionItems | undefined;
  setItems(value?: DataRetentionItems): DataRetention;
  hasItems(): boolean;
  clearItems(): DataRetention;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DataRetention.AsObject;
  static toObject(includeInstance: boolean, msg: DataRetention): DataRetention.AsObject;
  static serializeBinaryToWriter(message: DataRetention, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DataRetention;
  static deserializeBinaryFromReader(message: DataRetention, reader: jspb.BinaryReader): DataRetention;
}

export namespace DataRetention {
  export type AsObject = {
    bytes?: DataRetentionBytes.AsObject;
    items?: DataRetentionItems.AsObject;
  };
}

export class DataRetentionBytes extends jspb.Message {
  getUsed(): number;
  setUsed(value: number): DataRetentionBytes;
  hasUsed(): boolean;
  clearUsed(): DataRetentionBytes;

  getCapacity(): number;
  setCapacity(value: number): DataRetentionBytes;
  hasCapacity(): boolean;
  clearCapacity(): DataRetentionBytes;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DataRetentionBytes.AsObject;
  static toObject(includeInstance: boolean, msg: DataRetentionBytes): DataRetentionBytes.AsObject;
  static serializeBinaryToWriter(message: DataRetentionBytes, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DataRetentionBytes;
  static deserializeBinaryFromReader(message: DataRetentionBytes, reader: jspb.BinaryReader): DataRetentionBytes;
}

export namespace DataRetentionBytes {
  export type AsObject = {
    used?: number;
    capacity?: number;
  };

  export enum UsedCase {
    _USED_NOT_SET = 0,
    USED = 1,
  }

  export enum CapacityCase {
    _CAPACITY_NOT_SET = 0,
    CAPACITY = 2,
  }
}

export class DataRetentionItems extends jspb.Message {
  getUsed(): number;
  setUsed(value: number): DataRetentionItems;
  hasUsed(): boolean;
  clearUsed(): DataRetentionItems;

  getCapacity(): number;
  setCapacity(value: number): DataRetentionItems;
  hasCapacity(): boolean;
  clearCapacity(): DataRetentionItems;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DataRetentionItems.AsObject;
  static toObject(includeInstance: boolean, msg: DataRetentionItems): DataRetentionItems.AsObject;
  static serializeBinaryToWriter(message: DataRetentionItems, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DataRetentionItems;
  static deserializeBinaryFromReader(message: DataRetentionItems, reader: jspb.BinaryReader): DataRetentionItems;
}

export namespace DataRetentionItems {
  export type AsObject = {
    used?: number;
    capacity?: number;
  };

  export enum UsedCase {
    _USED_NOT_SET = 0,
    USED = 1,
  }

  export enum CapacityCase {
    _CAPACITY_NOT_SET = 0,
    CAPACITY = 2,
  }
}

export class DataRetentionSupport extends jspb.Message {
  getCanPurge(): boolean;
  setCanPurge(value: boolean): DataRetentionSupport;

  getCanCompact(): boolean;
  setCanCompact(value: boolean): DataRetentionSupport;

  getItemName(): string;
  setItemName(value: string): DataRetentionSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DataRetentionSupport.AsObject;
  static toObject(includeInstance: boolean, msg: DataRetentionSupport): DataRetentionSupport.AsObject;
  static serializeBinaryToWriter(message: DataRetentionSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DataRetentionSupport;
  static deserializeBinaryFromReader(message: DataRetentionSupport, reader: jspb.BinaryReader): DataRetentionSupport;
}

export namespace DataRetentionSupport {
  export type AsObject = {
    canPurge: boolean;
    canCompact: boolean;
    itemName: string;
  };
}

export class GetDataRetentionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetDataRetentionRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetDataRetentionRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetDataRetentionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDataRetentionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetDataRetentionRequest): GetDataRetentionRequest.AsObject;
  static serializeBinaryToWriter(message: GetDataRetentionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDataRetentionRequest;
  static deserializeBinaryFromReader(message: GetDataRetentionRequest, reader: jspb.BinaryReader): GetDataRetentionRequest;
}

export namespace GetDataRetentionRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullDataRetentionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullDataRetentionRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullDataRetentionRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullDataRetentionRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullDataRetentionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullDataRetentionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullDataRetentionRequest): PullDataRetentionRequest.AsObject;
  static serializeBinaryToWriter(message: PullDataRetentionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullDataRetentionRequest;
  static deserializeBinaryFromReader(message: PullDataRetentionRequest, reader: jspb.BinaryReader): PullDataRetentionRequest;
}

export namespace PullDataRetentionRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullDataRetentionResponse extends jspb.Message {
  getChangesList(): Array<PullDataRetentionResponse.Change>;
  setChangesList(value: Array<PullDataRetentionResponse.Change>): PullDataRetentionResponse;
  clearChangesList(): PullDataRetentionResponse;
  addChanges(value?: PullDataRetentionResponse.Change, index?: number): PullDataRetentionResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullDataRetentionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullDataRetentionResponse): PullDataRetentionResponse.AsObject;
  static serializeBinaryToWriter(message: PullDataRetentionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullDataRetentionResponse;
  static deserializeBinaryFromReader(message: PullDataRetentionResponse, reader: jspb.BinaryReader): PullDataRetentionResponse;
}

export namespace PullDataRetentionResponse {
  export type AsObject = {
    changesList: Array<PullDataRetentionResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getDataRetention(): DataRetention | undefined;
    setDataRetention(value?: DataRetention): Change;
    hasDataRetention(): boolean;
    clearDataRetention(): Change;

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
      dataRetention?: DataRetention.AsObject;
    };
  }

}

export class PurgeDataRetentionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PurgeDataRetentionRequest;

  getBefore(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setBefore(value?: google_protobuf_timestamp_pb.Timestamp): PurgeDataRetentionRequest;
  hasBefore(): boolean;
  clearBefore(): PurgeDataRetentionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PurgeDataRetentionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PurgeDataRetentionRequest): PurgeDataRetentionRequest.AsObject;
  static serializeBinaryToWriter(message: PurgeDataRetentionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PurgeDataRetentionRequest;
  static deserializeBinaryFromReader(message: PurgeDataRetentionRequest, reader: jspb.BinaryReader): PurgeDataRetentionRequest;
}

export namespace PurgeDataRetentionRequest {
  export type AsObject = {
    name: string;
    before?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };

  export enum BeforeCase {
    _BEFORE_NOT_SET = 0,
    BEFORE = 2,
  }
}

export class PurgeDataRetentionResponse extends jspb.Message {
  getFreedItemCount(): number;
  setFreedItemCount(value: number): PurgeDataRetentionResponse;
  hasFreedItemCount(): boolean;
  clearFreedItemCount(): PurgeDataRetentionResponse;

  getFreedByteCount(): number;
  setFreedByteCount(value: number): PurgeDataRetentionResponse;
  hasFreedByteCount(): boolean;
  clearFreedByteCount(): PurgeDataRetentionResponse;

  getMessage(): string;
  setMessage(value: string): PurgeDataRetentionResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PurgeDataRetentionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PurgeDataRetentionResponse): PurgeDataRetentionResponse.AsObject;
  static serializeBinaryToWriter(message: PurgeDataRetentionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PurgeDataRetentionResponse;
  static deserializeBinaryFromReader(message: PurgeDataRetentionResponse, reader: jspb.BinaryReader): PurgeDataRetentionResponse;
}

export namespace PurgeDataRetentionResponse {
  export type AsObject = {
    freedItemCount?: number;
    freedByteCount?: number;
    message: string;
  };

  export enum FreedItemCountCase {
    _FREED_ITEM_COUNT_NOT_SET = 0,
    FREED_ITEM_COUNT = 1,
  }

  export enum FreedByteCountCase {
    _FREED_BYTE_COUNT_NOT_SET = 0,
    FREED_BYTE_COUNT = 2,
  }
}

export class CompactDataRetentionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CompactDataRetentionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CompactDataRetentionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CompactDataRetentionRequest): CompactDataRetentionRequest.AsObject;
  static serializeBinaryToWriter(message: CompactDataRetentionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CompactDataRetentionRequest;
  static deserializeBinaryFromReader(message: CompactDataRetentionRequest, reader: jspb.BinaryReader): CompactDataRetentionRequest;
}

export namespace CompactDataRetentionRequest {
  export type AsObject = {
    name: string;
  };
}

export class CompactDataRetentionResponse extends jspb.Message {
  getFreedByteCount(): number;
  setFreedByteCount(value: number): CompactDataRetentionResponse;
  hasFreedByteCount(): boolean;
  clearFreedByteCount(): CompactDataRetentionResponse;

  getMessage(): string;
  setMessage(value: string): CompactDataRetentionResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CompactDataRetentionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: CompactDataRetentionResponse): CompactDataRetentionResponse.AsObject;
  static serializeBinaryToWriter(message: CompactDataRetentionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CompactDataRetentionResponse;
  static deserializeBinaryFromReader(message: CompactDataRetentionResponse, reader: jspb.BinaryReader): CompactDataRetentionResponse;
}

export namespace CompactDataRetentionResponse {
  export type AsObject = {
    freedByteCount?: number;
    message: string;
  };

  export enum FreedByteCountCase {
    _FREED_BYTE_COUNT_NOT_SET = 0,
    FREED_BYTE_COUNT = 1,
  }
}

export class DescribeDataRetentionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeDataRetentionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeDataRetentionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeDataRetentionRequest): DescribeDataRetentionRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeDataRetentionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeDataRetentionRequest;
  static deserializeBinaryFromReader(message: DescribeDataRetentionRequest, reader: jspb.BinaryReader): DescribeDataRetentionRequest;
}

export namespace DescribeDataRetentionRequest {
  export type AsObject = {
    name: string;
  };
}

