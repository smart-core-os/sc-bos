import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"


export class ListUnlockablesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListUnlockablesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListUnlockablesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListUnlockablesRequest): ListUnlockablesRequest.AsObject;
  static serializeBinaryToWriter(message: ListUnlockablesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListUnlockablesRequest;
  static deserializeBinaryFromReader(message: ListUnlockablesRequest, reader: jspb.BinaryReader): ListUnlockablesRequest;
}

export namespace ListUnlockablesRequest {
  export type AsObject = {
    name: string;
  };
}

export class ListUnlockablesResponse extends jspb.Message {
  getUnlockableBanksList(): Array<UnlockableBank>;
  setUnlockableBanksList(value: Array<UnlockableBank>): ListUnlockablesResponse;
  clearUnlockableBanksList(): ListUnlockablesResponse;
  addUnlockableBanks(value?: UnlockableBank, index?: number): UnlockableBank;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListUnlockablesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListUnlockablesResponse): ListUnlockablesResponse.AsObject;
  static serializeBinaryToWriter(message: ListUnlockablesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListUnlockablesResponse;
  static deserializeBinaryFromReader(message: ListUnlockablesResponse, reader: jspb.BinaryReader): ListUnlockablesResponse;
}

export namespace ListUnlockablesResponse {
  export type AsObject = {
    unlockableBanksList: Array<UnlockableBank.AsObject>;
  };
}

export class ListUnlockableHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListUnlockableHistoryRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListUnlockableHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListUnlockableHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListUnlockableHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListUnlockableHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListUnlockableHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListUnlockableHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListUnlockableHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListUnlockableHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListUnlockableHistoryRequest): ListUnlockableHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListUnlockableHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListUnlockableHistoryRequest;
  static deserializeBinaryFromReader(message: ListUnlockableHistoryRequest, reader: jspb.BinaryReader): ListUnlockableHistoryRequest;
}

export namespace ListUnlockableHistoryRequest {
  export type AsObject = {
    name: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListUnlockableHistoryResponse extends jspb.Message {
  getUnlockableRecordsList(): Array<UnlockableRecord>;
  setUnlockableRecordsList(value: Array<UnlockableRecord>): ListUnlockableHistoryResponse;
  clearUnlockableRecordsList(): ListUnlockableHistoryResponse;
  addUnlockableRecords(value?: UnlockableRecord, index?: number): UnlockableRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListUnlockableHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListUnlockableHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListUnlockableHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListUnlockableHistoryResponse): ListUnlockableHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListUnlockableHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListUnlockableHistoryResponse;
  static deserializeBinaryFromReader(message: ListUnlockableHistoryResponse, reader: jspb.BinaryReader): ListUnlockableHistoryResponse;
}

export namespace ListUnlockableHistoryResponse {
  export type AsObject = {
    unlockableRecordsList: Array<UnlockableRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class UnlockableRecord extends jspb.Message {
  getUnlockable(): Unlockable | undefined;
  setUnlockable(value?: Unlockable): UnlockableRecord;
  hasUnlockable(): boolean;
  clearUnlockable(): UnlockableRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): UnlockableRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): UnlockableRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UnlockableRecord.AsObject;
  static toObject(includeInstance: boolean, msg: UnlockableRecord): UnlockableRecord.AsObject;
  static serializeBinaryToWriter(message: UnlockableRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UnlockableRecord;
  static deserializeBinaryFromReader(message: UnlockableRecord, reader: jspb.BinaryReader): UnlockableRecord;
}

export namespace UnlockableRecord {
  export type AsObject = {
    unlockable?: Unlockable.AsObject;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class Unlockable extends jspb.Message {
  getId(): string;
  setId(value: string): Unlockable;

  getTitle(): string;
  setTitle(value: string): Unlockable;

  getUnlockableBankId(): string;
  setUnlockableBankId(value: string): Unlockable;

  getAllocations(): number;
  setAllocations(value: number): Unlockable;

  getIsUsable(): boolean;
  setIsUsable(value: boolean): Unlockable;

  getLastUpdated(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastUpdated(value?: google_protobuf_timestamp_pb.Timestamp): Unlockable;
  hasLastUpdated(): boolean;
  clearLastUpdated(): Unlockable;

  getIsLocked(): boolean;
  setIsLocked(value: boolean): Unlockable;

  getLocationId(): string;
  setLocationId(value: string): Unlockable;
  hasLocationId(): boolean;
  clearLocationId(): Unlockable;

  getSharedToUser(): boolean;
  setSharedToUser(value: boolean): Unlockable;
  hasSharedToUser(): boolean;
  clearSharedToUser(): Unlockable;

  getIsShared(): boolean;
  setIsShared(value: boolean): Unlockable;
  hasIsShared(): boolean;
  clearIsShared(): Unlockable;

  getIsShareable(): boolean;
  setIsShareable(value: boolean): Unlockable;
  hasIsShareable(): boolean;
  clearIsShareable(): Unlockable;

  getSharedBy(): string;
  setSharedBy(value: string): Unlockable;
  hasSharedBy(): boolean;
  clearSharedBy(): Unlockable;

  getPhysicalPin(): string;
  setPhysicalPin(value: string): Unlockable;
  hasPhysicalPin(): boolean;
  clearPhysicalPin(): Unlockable;

  getExpiresDateTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setExpiresDateTime(value?: google_protobuf_timestamp_pb.Timestamp): Unlockable;
  hasExpiresDateTime(): boolean;
  clearExpiresDateTime(): Unlockable;

  getStartDateTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartDateTime(value?: google_protobuf_timestamp_pb.Timestamp): Unlockable;
  hasStartDateTime(): boolean;
  clearStartDateTime(): Unlockable;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Unlockable.AsObject;
  static toObject(includeInstance: boolean, msg: Unlockable): Unlockable.AsObject;
  static serializeBinaryToWriter(message: Unlockable, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Unlockable;
  static deserializeBinaryFromReader(message: Unlockable, reader: jspb.BinaryReader): Unlockable;
}

export namespace Unlockable {
  export type AsObject = {
    id: string;
    title: string;
    unlockableBankId: string;
    allocations: number;
    isUsable: boolean;
    lastUpdated?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    isLocked: boolean;
    locationId?: string;
    sharedToUser?: boolean;
    isShared?: boolean;
    isShareable?: boolean;
    sharedBy?: string;
    physicalPin?: string;
    expiresDateTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    startDateTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };

  export enum LocationIdCase {
    _LOCATION_ID_NOT_SET = 0,
    LOCATION_ID = 10,
  }

  export enum SharedToUserCase {
    _SHARED_TO_USER_NOT_SET = 0,
    SHARED_TO_USER = 11,
  }

  export enum IsSharedCase {
    _IS_SHARED_NOT_SET = 0,
    IS_SHARED = 12,
  }

  export enum IsShareableCase {
    _IS_SHAREABLE_NOT_SET = 0,
    IS_SHAREABLE = 13,
  }

  export enum SharedByCase {
    _SHARED_BY_NOT_SET = 0,
    SHARED_BY = 14,
  }

  export enum PhysicalPinCase {
    _PHYSICAL_PIN_NOT_SET = 0,
    PHYSICAL_PIN = 15,
  }

  export enum ExpiresDateTimeCase {
    _EXPIRES_DATE_TIME_NOT_SET = 0,
    EXPIRES_DATE_TIME = 16,
  }

  export enum StartDateTimeCase {
    _START_DATE_TIME_NOT_SET = 0,
    START_DATE_TIME = 17,
  }
}

export class UnlockableBank extends jspb.Message {
  getId(): string;
  setId(value: string): UnlockableBank;

  getTitle(): string;
  setTitle(value: string): UnlockableBank;

  getLocationId(): string;
  setLocationId(value: string): UnlockableBank;

  getUnlockablesList(): Array<Unlockable>;
  setUnlockablesList(value: Array<Unlockable>): UnlockableBank;
  clearUnlockablesList(): UnlockableBank;
  addUnlockables(value?: Unlockable, index?: number): Unlockable;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UnlockableBank.AsObject;
  static toObject(includeInstance: boolean, msg: UnlockableBank): UnlockableBank.AsObject;
  static serializeBinaryToWriter(message: UnlockableBank, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UnlockableBank;
  static deserializeBinaryFromReader(message: UnlockableBank, reader: jspb.BinaryReader): UnlockableBank;
}

export namespace UnlockableBank {
  export type AsObject = {
    id: string;
    title: string;
    locationId: string;
    unlockablesList: Array<Unlockable.AsObject>;
  };
}

export class UnlockUnlockableRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UnlockUnlockableRequest;

  getUnlockableId(): string;
  setUnlockableId(value: string): UnlockUnlockableRequest;

  getUserId(): string;
  setUserId(value: string): UnlockUnlockableRequest;

  getOpenTimeSeconds(): number;
  setOpenTimeSeconds(value: number): UnlockUnlockableRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UnlockUnlockableRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UnlockUnlockableRequest): UnlockUnlockableRequest.AsObject;
  static serializeBinaryToWriter(message: UnlockUnlockableRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UnlockUnlockableRequest;
  static deserializeBinaryFromReader(message: UnlockUnlockableRequest, reader: jspb.BinaryReader): UnlockUnlockableRequest;
}

export namespace UnlockUnlockableRequest {
  export type AsObject = {
    name: string;
    unlockableId: string;
    userId: string;
    openTimeSeconds: number;
  };
}

export class UnlockUnlockableResponse extends jspb.Message {
  getUnlockableId(): string;
  setUnlockableId(value: string): UnlockUnlockableResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UnlockUnlockableResponse.AsObject;
  static toObject(includeInstance: boolean, msg: UnlockUnlockableResponse): UnlockUnlockableResponse.AsObject;
  static serializeBinaryToWriter(message: UnlockUnlockableResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UnlockUnlockableResponse;
  static deserializeBinaryFromReader(message: UnlockUnlockableResponse, reader: jspb.BinaryReader): UnlockUnlockableResponse;
}

export namespace UnlockUnlockableResponse {
  export type AsObject = {
    unlockableId: string;
  };
}

export class LockUnlockableRequest extends jspb.Message {
  getName(): string;
  setName(value: string): LockUnlockableRequest;

  getUnlockableId(): string;
  setUnlockableId(value: string): LockUnlockableRequest;

  getUserId(): string;
  setUserId(value: string): LockUnlockableRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LockUnlockableRequest.AsObject;
  static toObject(includeInstance: boolean, msg: LockUnlockableRequest): LockUnlockableRequest.AsObject;
  static serializeBinaryToWriter(message: LockUnlockableRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LockUnlockableRequest;
  static deserializeBinaryFromReader(message: LockUnlockableRequest, reader: jspb.BinaryReader): LockUnlockableRequest;
}

export namespace LockUnlockableRequest {
  export type AsObject = {
    name: string;
    unlockableId: string;
    userId: string;
  };
}

export class LockUnlockableResponse extends jspb.Message {
  getUnlockableId(): string;
  setUnlockableId(value: string): LockUnlockableResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LockUnlockableResponse.AsObject;
  static toObject(includeInstance: boolean, msg: LockUnlockableResponse): LockUnlockableResponse.AsObject;
  static serializeBinaryToWriter(message: LockUnlockableResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LockUnlockableResponse;
  static deserializeBinaryFromReader(message: LockUnlockableResponse, reader: jspb.BinaryReader): LockUnlockableResponse;
}

export namespace LockUnlockableResponse {
  export type AsObject = {
    unlockableId: string;
  };
}

