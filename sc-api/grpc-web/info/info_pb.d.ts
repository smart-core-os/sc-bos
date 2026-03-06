import * as jspb from 'google-protobuf'

import * as types_change_pb from '../types/change_pb'; // proto import: "types/change.proto"


export class ListDevicesRequest extends jspb.Message {
  getDepth(): number;
  setDepth(value: number): ListDevicesRequest;

  getPageSize(): number;
  setPageSize(value: number): ListDevicesRequest;

  getPageToken(): string;
  setPageToken(value: string): ListDevicesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListDevicesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListDevicesRequest): ListDevicesRequest.AsObject;
  static serializeBinaryToWriter(message: ListDevicesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListDevicesRequest;
  static deserializeBinaryFromReader(message: ListDevicesRequest, reader: jspb.BinaryReader): ListDevicesRequest;
}

export namespace ListDevicesRequest {
  export type AsObject = {
    depth: number;
    pageSize: number;
    pageToken: string;
  };
}

export class ListDevicesResponse extends jspb.Message {
  getDevicesList(): Array<Device>;
  setDevicesList(value: Array<Device>): ListDevicesResponse;
  clearDevicesList(): ListDevicesResponse;
  addDevices(value?: Device, index?: number): Device;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListDevicesResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListDevicesResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListDevicesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListDevicesResponse): ListDevicesResponse.AsObject;
  static serializeBinaryToWriter(message: ListDevicesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListDevicesResponse;
  static deserializeBinaryFromReader(message: ListDevicesResponse, reader: jspb.BinaryReader): ListDevicesResponse;
}

export namespace ListDevicesResponse {
  export type AsObject = {
    devicesList: Array<Device.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class PullDevicesRequest extends jspb.Message {
  getDepth(): number;
  setDepth(value: number): PullDevicesRequest;

  getSync(): boolean;
  setSync(value: boolean): PullDevicesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullDevicesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullDevicesRequest): PullDevicesRequest.AsObject;
  static serializeBinaryToWriter(message: PullDevicesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullDevicesRequest;
  static deserializeBinaryFromReader(message: PullDevicesRequest, reader: jspb.BinaryReader): PullDevicesRequest;
}

export namespace PullDevicesRequest {
  export type AsObject = {
    depth: number;
    sync: boolean;
  };
}

export class PullDevicesResponse extends jspb.Message {
  getChangesList(): Array<PullDevicesResponse.Change>;
  setChangesList(value: Array<PullDevicesResponse.Change>): PullDevicesResponse;
  clearChangesList(): PullDevicesResponse;
  addChanges(value?: PullDevicesResponse.Change, index?: number): PullDevicesResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullDevicesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullDevicesResponse): PullDevicesResponse.AsObject;
  static serializeBinaryToWriter(message: PullDevicesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullDevicesResponse;
  static deserializeBinaryFromReader(message: PullDevicesResponse, reader: jspb.BinaryReader): PullDevicesResponse;
}

export namespace PullDevicesResponse {
  export type AsObject = {
    changesList: Array<PullDevicesResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getType(): types_change_pb.ChangeType;
    setType(value: types_change_pb.ChangeType): Change;

    getNewValue(): Device | undefined;
    setNewValue(value?: Device): Change;
    hasNewValue(): boolean;
    clearNewValue(): Change;

    getOldValue(): Device | undefined;
    setOldValue(value?: Device): Change;
    hasOldValue(): boolean;
    clearOldValue(): Change;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Change.AsObject;
    static toObject(includeInstance: boolean, msg: Change): Change.AsObject;
    static serializeBinaryToWriter(message: Change, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Change;
    static deserializeBinaryFromReader(message: Change, reader: jspb.BinaryReader): Change;
  }

  export namespace Change {
    export type AsObject = {
      type: types_change_pb.ChangeType;
      newValue?: Device.AsObject;
      oldValue?: Device.AsObject;
    };
  }

}

export class Device extends jspb.Message {
  getName(): string;
  setName(value: string): Device;

  getTraitsList(): Array<Trait>;
  setTraitsList(value: Array<Trait>): Device;
  clearTraitsList(): Device;
  addTraits(value?: Trait, index?: number): Trait;

  getOwner(): Device | undefined;
  setOwner(value?: Device): Device;
  hasOwner(): boolean;
  clearOwner(): Device;

  getClient(): GrpcClientOptions | undefined;
  setClient(value?: GrpcClientOptions): Device;
  hasClient(): boolean;
  clearClient(): Device;

  getTitle(): string;
  setTitle(value: string): Device;

  getDisplayName(): string;
  setDisplayName(value: string): Device;

  getDescription(): string;
  setDescription(value: string): Device;

  getLabelsMap(): jspb.Map<string, string>;
  clearLabelsMap(): Device;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Device.AsObject;
  static toObject(includeInstance: boolean, msg: Device): Device.AsObject;
  static serializeBinaryToWriter(message: Device, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Device;
  static deserializeBinaryFromReader(message: Device, reader: jspb.BinaryReader): Device;
}

export namespace Device {
  export type AsObject = {
    name: string;
    traitsList: Array<Trait.AsObject>;
    owner?: Device.AsObject;
    client?: GrpcClientOptions.AsObject;
    title: string;
    displayName: string;
    description: string;
    labelsMap: Array<[string, string]>;
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

export class GrpcClientOptions extends jspb.Message {
  getAuthority(): string;
  setAuthority(value: string): GrpcClientOptions;

  getClientcert(): Uint8Array | string;
  getClientcert_asU8(): Uint8Array;
  getClientcert_asB64(): string;
  setClientcert(value: Uint8Array | string): GrpcClientOptions;

  getClientkey(): Uint8Array | string;
  getClientkey_asU8(): Uint8Array;
  getClientkey_asB64(): string;
  setClientkey(value: Uint8Array | string): GrpcClientOptions;

  getClientca(): Uint8Array | string;
  getClientca_asU8(): Uint8Array;
  getClientca_asB64(): string;
  setClientca(value: Uint8Array | string): GrpcClientOptions;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GrpcClientOptions.AsObject;
  static toObject(includeInstance: boolean, msg: GrpcClientOptions): GrpcClientOptions.AsObject;
  static serializeBinaryToWriter(message: GrpcClientOptions, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GrpcClientOptions;
  static deserializeBinaryFromReader(message: GrpcClientOptions, reader: jspb.BinaryReader): GrpcClientOptions;
}

export namespace GrpcClientOptions {
  export type AsObject = {
    authority: string;
    clientcert: Uint8Array | string;
    clientkey: Uint8Array | string;
    clientca: Uint8Array | string;
  };
}

