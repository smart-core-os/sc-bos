import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class ResourceUse extends jspb.Message {
  getCpu(): CpuUse | undefined;
  setCpu(value?: CpuUse): ResourceUse;
  hasCpu(): boolean;
  clearCpu(): ResourceUse;

  getMemory(): MemoryUse | undefined;
  setMemory(value?: MemoryUse): ResourceUse;
  hasMemory(): boolean;
  clearMemory(): ResourceUse;

  getDisksList(): Array<DiskUse>;
  setDisksList(value: Array<DiskUse>): ResourceUse;
  clearDisksList(): ResourceUse;
  addDisks(value?: DiskUse, index?: number): DiskUse;

  getNetwork(): NetworkUse | undefined;
  setNetwork(value?: NetworkUse): ResourceUse;
  hasNetwork(): boolean;
  clearNetwork(): ResourceUse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ResourceUse.AsObject;
  static toObject(includeInstance: boolean, msg: ResourceUse): ResourceUse.AsObject;
  static serializeBinaryToWriter(message: ResourceUse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ResourceUse;
  static deserializeBinaryFromReader(message: ResourceUse, reader: jspb.BinaryReader): ResourceUse;
}

export namespace ResourceUse {
  export type AsObject = {
    cpu?: CpuUse.AsObject;
    memory?: MemoryUse.AsObject;
    disksList: Array<DiskUse.AsObject>;
    network?: NetworkUse.AsObject;
  };
}

export class CpuUse extends jspb.Message {
  getPercentUtilised(): number;
  setPercentUtilised(value: number): CpuUse;
  hasPercentUtilised(): boolean;
  clearPercentUtilised(): CpuUse;

  getCorePercentList(): Array<number>;
  setCorePercentList(value: Array<number>): CpuUse;
  clearCorePercentList(): CpuUse;
  addCorePercent(value: number, index?: number): CpuUse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CpuUse.AsObject;
  static toObject(includeInstance: boolean, msg: CpuUse): CpuUse.AsObject;
  static serializeBinaryToWriter(message: CpuUse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CpuUse;
  static deserializeBinaryFromReader(message: CpuUse, reader: jspb.BinaryReader): CpuUse;
}

export namespace CpuUse {
  export type AsObject = {
    percentUtilised?: number;
    corePercentList: Array<number>;
  };

  export enum PercentUtilisedCase {
    _PERCENT_UTILISED_NOT_SET = 0,
    PERCENT_UTILISED = 1,
  }
}

export class MemoryUse extends jspb.Message {
  getUsedBytes(): number;
  setUsedBytes(value: number): MemoryUse;
  hasUsedBytes(): boolean;
  clearUsedBytes(): MemoryUse;

  getTotalBytes(): number;
  setTotalBytes(value: number): MemoryUse;
  hasTotalBytes(): boolean;
  clearTotalBytes(): MemoryUse;

  getPercentUsed(): number;
  setPercentUsed(value: number): MemoryUse;
  hasPercentUsed(): boolean;
  clearPercentUsed(): MemoryUse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MemoryUse.AsObject;
  static toObject(includeInstance: boolean, msg: MemoryUse): MemoryUse.AsObject;
  static serializeBinaryToWriter(message: MemoryUse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MemoryUse;
  static deserializeBinaryFromReader(message: MemoryUse, reader: jspb.BinaryReader): MemoryUse;
}

export namespace MemoryUse {
  export type AsObject = {
    usedBytes?: number;
    totalBytes?: number;
    percentUsed?: number;
  };

  export enum UsedBytesCase {
    _USED_BYTES_NOT_SET = 0,
    USED_BYTES = 1,
  }

  export enum TotalBytesCase {
    _TOTAL_BYTES_NOT_SET = 0,
    TOTAL_BYTES = 2,
  }

  export enum PercentUsedCase {
    _PERCENT_USED_NOT_SET = 0,
    PERCENT_USED = 3,
  }
}

export class DiskUse extends jspb.Message {
  getMountPoint(): string;
  setMountPoint(value: string): DiskUse;

  getUsedBytes(): number;
  setUsedBytes(value: number): DiskUse;
  hasUsedBytes(): boolean;
  clearUsedBytes(): DiskUse;

  getFreeBytes(): number;
  setFreeBytes(value: number): DiskUse;
  hasFreeBytes(): boolean;
  clearFreeBytes(): DiskUse;

  getTotalBytes(): number;
  setTotalBytes(value: number): DiskUse;
  hasTotalBytes(): boolean;
  clearTotalBytes(): DiskUse;

  getPercentUsed(): number;
  setPercentUsed(value: number): DiskUse;
  hasPercentUsed(): boolean;
  clearPercentUsed(): DiskUse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DiskUse.AsObject;
  static toObject(includeInstance: boolean, msg: DiskUse): DiskUse.AsObject;
  static serializeBinaryToWriter(message: DiskUse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DiskUse;
  static deserializeBinaryFromReader(message: DiskUse, reader: jspb.BinaryReader): DiskUse;
}

export namespace DiskUse {
  export type AsObject = {
    mountPoint: string;
    usedBytes?: number;
    freeBytes?: number;
    totalBytes?: number;
    percentUsed?: number;
  };

  export enum UsedBytesCase {
    _USED_BYTES_NOT_SET = 0,
    USED_BYTES = 2,
  }

  export enum FreeBytesCase {
    _FREE_BYTES_NOT_SET = 0,
    FREE_BYTES = 3,
  }

  export enum TotalBytesCase {
    _TOTAL_BYTES_NOT_SET = 0,
    TOTAL_BYTES = 4,
  }

  export enum PercentUsedCase {
    _PERCENT_USED_NOT_SET = 0,
    PERCENT_USED = 5,
  }
}

export class NetworkUse extends jspb.Message {
  getConnectionsEstablished(): number;
  setConnectionsEstablished(value: number): NetworkUse;
  hasConnectionsEstablished(): boolean;
  clearConnectionsEstablished(): NetworkUse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NetworkUse.AsObject;
  static toObject(includeInstance: boolean, msg: NetworkUse): NetworkUse.AsObject;
  static serializeBinaryToWriter(message: NetworkUse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NetworkUse;
  static deserializeBinaryFromReader(message: NetworkUse, reader: jspb.BinaryReader): NetworkUse;
}

export namespace NetworkUse {
  export type AsObject = {
    connectionsEstablished?: number;
  };

  export enum ConnectionsEstablishedCase {
    _CONNECTIONS_ESTABLISHED_NOT_SET = 0,
    CONNECTIONS_ESTABLISHED = 1,
  }
}

export class GetResourceUseRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetResourceUseRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetResourceUseRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetResourceUseRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetResourceUseRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetResourceUseRequest): GetResourceUseRequest.AsObject;
  static serializeBinaryToWriter(message: GetResourceUseRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetResourceUseRequest;
  static deserializeBinaryFromReader(message: GetResourceUseRequest, reader: jspb.BinaryReader): GetResourceUseRequest;
}

export namespace GetResourceUseRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullResourceUseRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullResourceUseRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullResourceUseRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullResourceUseRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullResourceUseRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullResourceUseRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullResourceUseRequest): PullResourceUseRequest.AsObject;
  static serializeBinaryToWriter(message: PullResourceUseRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullResourceUseRequest;
  static deserializeBinaryFromReader(message: PullResourceUseRequest, reader: jspb.BinaryReader): PullResourceUseRequest;
}

export namespace PullResourceUseRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullResourceUseResponse extends jspb.Message {
  getChangesList(): Array<PullResourceUseResponse.Change>;
  setChangesList(value: Array<PullResourceUseResponse.Change>): PullResourceUseResponse;
  clearChangesList(): PullResourceUseResponse;
  addChanges(value?: PullResourceUseResponse.Change, index?: number): PullResourceUseResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullResourceUseResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullResourceUseResponse): PullResourceUseResponse.AsObject;
  static serializeBinaryToWriter(message: PullResourceUseResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullResourceUseResponse;
  static deserializeBinaryFromReader(message: PullResourceUseResponse, reader: jspb.BinaryReader): PullResourceUseResponse;
}

export namespace PullResourceUseResponse {
  export type AsObject = {
    changesList: Array<PullResourceUseResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getResourceUse(): ResourceUse | undefined;
    setResourceUse(value?: ResourceUse): Change;
    hasResourceUse(): boolean;
    clearResourceUse(): Change;

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
      resourceUse?: ResourceUse.AsObject;
    };
  }

}

