import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class ResourceUtilisation extends jspb.Message {
  getCpu(): CpuUtilisation | undefined;
  setCpu(value?: CpuUtilisation): ResourceUtilisation;
  hasCpu(): boolean;
  clearCpu(): ResourceUtilisation;

  getMemory(): MemoryUtilisation | undefined;
  setMemory(value?: MemoryUtilisation): ResourceUtilisation;
  hasMemory(): boolean;
  clearMemory(): ResourceUtilisation;

  getDisksList(): Array<DiskUtilisation>;
  setDisksList(value: Array<DiskUtilisation>): ResourceUtilisation;
  clearDisksList(): ResourceUtilisation;
  addDisks(value?: DiskUtilisation, index?: number): DiskUtilisation;

  getNetwork(): NetworkUtilisation | undefined;
  setNetwork(value?: NetworkUtilisation): ResourceUtilisation;
  hasNetwork(): boolean;
  clearNetwork(): ResourceUtilisation;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ResourceUtilisation.AsObject;
  static toObject(includeInstance: boolean, msg: ResourceUtilisation): ResourceUtilisation.AsObject;
  static serializeBinaryToWriter(message: ResourceUtilisation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ResourceUtilisation;
  static deserializeBinaryFromReader(message: ResourceUtilisation, reader: jspb.BinaryReader): ResourceUtilisation;
}

export namespace ResourceUtilisation {
  export type AsObject = {
    cpu?: CpuUtilisation.AsObject;
    memory?: MemoryUtilisation.AsObject;
    disksList: Array<DiskUtilisation.AsObject>;
    network?: NetworkUtilisation.AsObject;
  };
}

export class CpuUtilisation extends jspb.Message {
  getPercentUtilised(): number;
  setPercentUtilised(value: number): CpuUtilisation;
  hasPercentUtilised(): boolean;
  clearPercentUtilised(): CpuUtilisation;

  getCorePercentList(): Array<number>;
  setCorePercentList(value: Array<number>): CpuUtilisation;
  clearCorePercentList(): CpuUtilisation;
  addCorePercent(value: number, index?: number): CpuUtilisation;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CpuUtilisation.AsObject;
  static toObject(includeInstance: boolean, msg: CpuUtilisation): CpuUtilisation.AsObject;
  static serializeBinaryToWriter(message: CpuUtilisation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CpuUtilisation;
  static deserializeBinaryFromReader(message: CpuUtilisation, reader: jspb.BinaryReader): CpuUtilisation;
}

export namespace CpuUtilisation {
  export type AsObject = {
    percentUtilised?: number;
    corePercentList: Array<number>;
  };

  export enum PercentUtilisedCase {
    _PERCENT_UTILISED_NOT_SET = 0,
    PERCENT_UTILISED = 1,
  }
}

export class MemoryUtilisation extends jspb.Message {
  getUsedBytes(): number;
  setUsedBytes(value: number): MemoryUtilisation;
  hasUsedBytes(): boolean;
  clearUsedBytes(): MemoryUtilisation;

  getTotalBytes(): number;
  setTotalBytes(value: number): MemoryUtilisation;
  hasTotalBytes(): boolean;
  clearTotalBytes(): MemoryUtilisation;

  getPercentUsed(): number;
  setPercentUsed(value: number): MemoryUtilisation;
  hasPercentUsed(): boolean;
  clearPercentUsed(): MemoryUtilisation;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MemoryUtilisation.AsObject;
  static toObject(includeInstance: boolean, msg: MemoryUtilisation): MemoryUtilisation.AsObject;
  static serializeBinaryToWriter(message: MemoryUtilisation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MemoryUtilisation;
  static deserializeBinaryFromReader(message: MemoryUtilisation, reader: jspb.BinaryReader): MemoryUtilisation;
}

export namespace MemoryUtilisation {
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

export class DiskUtilisation extends jspb.Message {
  getMountPoint(): string;
  setMountPoint(value: string): DiskUtilisation;

  getUsedBytes(): number;
  setUsedBytes(value: number): DiskUtilisation;
  hasUsedBytes(): boolean;
  clearUsedBytes(): DiskUtilisation;

  getFreeBytes(): number;
  setFreeBytes(value: number): DiskUtilisation;
  hasFreeBytes(): boolean;
  clearFreeBytes(): DiskUtilisation;

  getTotalBytes(): number;
  setTotalBytes(value: number): DiskUtilisation;
  hasTotalBytes(): boolean;
  clearTotalBytes(): DiskUtilisation;

  getPercentUsed(): number;
  setPercentUsed(value: number): DiskUtilisation;
  hasPercentUsed(): boolean;
  clearPercentUsed(): DiskUtilisation;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DiskUtilisation.AsObject;
  static toObject(includeInstance: boolean, msg: DiskUtilisation): DiskUtilisation.AsObject;
  static serializeBinaryToWriter(message: DiskUtilisation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DiskUtilisation;
  static deserializeBinaryFromReader(message: DiskUtilisation, reader: jspb.BinaryReader): DiskUtilisation;
}

export namespace DiskUtilisation {
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

export class NetworkUtilisation extends jspb.Message {
  getConnectionsEstablished(): number;
  setConnectionsEstablished(value: number): NetworkUtilisation;
  hasConnectionsEstablished(): boolean;
  clearConnectionsEstablished(): NetworkUtilisation;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NetworkUtilisation.AsObject;
  static toObject(includeInstance: boolean, msg: NetworkUtilisation): NetworkUtilisation.AsObject;
  static serializeBinaryToWriter(message: NetworkUtilisation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NetworkUtilisation;
  static deserializeBinaryFromReader(message: NetworkUtilisation, reader: jspb.BinaryReader): NetworkUtilisation;
}

export namespace NetworkUtilisation {
  export type AsObject = {
    connectionsEstablished?: number;
  };

  export enum ConnectionsEstablishedCase {
    _CONNECTIONS_ESTABLISHED_NOT_SET = 0,
    CONNECTIONS_ESTABLISHED = 1,
  }
}

export class GetResourceUtilisationRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetResourceUtilisationRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetResourceUtilisationRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetResourceUtilisationRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetResourceUtilisationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetResourceUtilisationRequest): GetResourceUtilisationRequest.AsObject;
  static serializeBinaryToWriter(message: GetResourceUtilisationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetResourceUtilisationRequest;
  static deserializeBinaryFromReader(message: GetResourceUtilisationRequest, reader: jspb.BinaryReader): GetResourceUtilisationRequest;
}

export namespace GetResourceUtilisationRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullResourceUtilisationRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullResourceUtilisationRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullResourceUtilisationRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullResourceUtilisationRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullResourceUtilisationRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullResourceUtilisationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullResourceUtilisationRequest): PullResourceUtilisationRequest.AsObject;
  static serializeBinaryToWriter(message: PullResourceUtilisationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullResourceUtilisationRequest;
  static deserializeBinaryFromReader(message: PullResourceUtilisationRequest, reader: jspb.BinaryReader): PullResourceUtilisationRequest;
}

export namespace PullResourceUtilisationRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullResourceUtilisationResponse extends jspb.Message {
  getChangesList(): Array<PullResourceUtilisationResponse.Change>;
  setChangesList(value: Array<PullResourceUtilisationResponse.Change>): PullResourceUtilisationResponse;
  clearChangesList(): PullResourceUtilisationResponse;
  addChanges(value?: PullResourceUtilisationResponse.Change, index?: number): PullResourceUtilisationResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullResourceUtilisationResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullResourceUtilisationResponse): PullResourceUtilisationResponse.AsObject;
  static serializeBinaryToWriter(message: PullResourceUtilisationResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullResourceUtilisationResponse;
  static deserializeBinaryFromReader(message: PullResourceUtilisationResponse, reader: jspb.BinaryReader): PullResourceUtilisationResponse;
}

export namespace PullResourceUtilisationResponse {
  export type AsObject = {
    changesList: Array<PullResourceUtilisationResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getResourceUtilisation(): ResourceUtilisation | undefined;
    setResourceUtilisation(value?: ResourceUtilisation): Change;
    hasResourceUtilisation(): boolean;
    clearResourceUtilisation(): Change;

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
      resourceUtilisation?: ResourceUtilisation.AsObject;
    };
  }

}

