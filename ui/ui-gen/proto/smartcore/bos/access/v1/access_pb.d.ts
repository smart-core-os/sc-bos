import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_actor_v1_actor_pb from '../../../../smartcore/bos/actor/v1/actor_pb'; // proto import: "smartcore/bos/actor/v1/actor.proto"


export class AccessAttempt extends jspb.Message {
  getGrant(): AccessAttempt.Grant;
  setGrant(value: AccessAttempt.Grant): AccessAttempt;

  getReason(): string;
  setReason(value: string): AccessAttempt;

  getActor(): smartcore_bos_actor_v1_actor_pb.Actor | undefined;
  setActor(value?: smartcore_bos_actor_v1_actor_pb.Actor): AccessAttempt;
  hasActor(): boolean;
  clearActor(): AccessAttempt;

  getAccessAttemptTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setAccessAttemptTime(value?: google_protobuf_timestamp_pb.Timestamp): AccessAttempt;
  hasAccessAttemptTime(): boolean;
  clearAccessAttemptTime(): AccessAttempt;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AccessAttempt.AsObject;
  static toObject(includeInstance: boolean, msg: AccessAttempt): AccessAttempt.AsObject;
  static serializeBinaryToWriter(message: AccessAttempt, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AccessAttempt;
  static deserializeBinaryFromReader(message: AccessAttempt, reader: jspb.BinaryReader): AccessAttempt;
}

export namespace AccessAttempt {
  export type AsObject = {
    grant: AccessAttempt.Grant;
    reason: string;
    actor?: smartcore_bos_actor_v1_actor_pb.Actor.AsObject;
    accessAttemptTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };

  export enum Grant {
    GRANT_UNKNOWN = 0,
    GRANTED = 1,
    DENIED = 2,
    PENDING = 3,
    ABORTED = 4,
    FORCED = 5,
    FAILED = 6,
    TAILGATE = 7,
  }
}

export class GetLastAccessAttemptRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetLastAccessAttemptRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetLastAccessAttemptRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetLastAccessAttemptRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetLastAccessAttemptRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetLastAccessAttemptRequest): GetLastAccessAttemptRequest.AsObject;
  static serializeBinaryToWriter(message: GetLastAccessAttemptRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetLastAccessAttemptRequest;
  static deserializeBinaryFromReader(message: GetLastAccessAttemptRequest, reader: jspb.BinaryReader): GetLastAccessAttemptRequest;
}

export namespace GetLastAccessAttemptRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullAccessAttemptsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullAccessAttemptsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullAccessAttemptsRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullAccessAttemptsRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullAccessAttemptsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullAccessAttemptsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullAccessAttemptsRequest): PullAccessAttemptsRequest.AsObject;
  static serializeBinaryToWriter(message: PullAccessAttemptsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullAccessAttemptsRequest;
  static deserializeBinaryFromReader(message: PullAccessAttemptsRequest, reader: jspb.BinaryReader): PullAccessAttemptsRequest;
}

export namespace PullAccessAttemptsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullAccessAttemptsResponse extends jspb.Message {
  getChangesList(): Array<PullAccessAttemptsResponse.Change>;
  setChangesList(value: Array<PullAccessAttemptsResponse.Change>): PullAccessAttemptsResponse;
  clearChangesList(): PullAccessAttemptsResponse;
  addChanges(value?: PullAccessAttemptsResponse.Change, index?: number): PullAccessAttemptsResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullAccessAttemptsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullAccessAttemptsResponse): PullAccessAttemptsResponse.AsObject;
  static serializeBinaryToWriter(message: PullAccessAttemptsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullAccessAttemptsResponse;
  static deserializeBinaryFromReader(message: PullAccessAttemptsResponse, reader: jspb.BinaryReader): PullAccessAttemptsResponse;
}

export namespace PullAccessAttemptsResponse {
  export type AsObject = {
    changesList: Array<PullAccessAttemptsResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getAccessAttempt(): AccessAttempt | undefined;
    setAccessAttempt(value?: AccessAttempt): Change;
    hasAccessAttempt(): boolean;
    clearAccessAttempt(): Change;

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
      accessAttempt?: AccessAttempt.AsObject;
    };
  }

}

export class AccessGrant extends jspb.Message {
  getId(): string;
  setId(value: string): AccessGrant;

  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): AccessGrant;
  hasStartTime(): boolean;
  clearStartTime(): AccessGrant;

  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): AccessGrant;
  hasEndTime(): boolean;
  clearEndTime(): AccessGrant;

  getPurpose(): string;
  setPurpose(value: string): AccessGrant;
  hasPurpose(): boolean;
  clearPurpose(): AccessGrant;

  getGrantee(): smartcore_bos_actor_v1_actor_pb.Actor | undefined;
  setGrantee(value?: smartcore_bos_actor_v1_actor_pb.Actor): AccessGrant;
  hasGrantee(): boolean;
  clearGrantee(): AccessGrant;

  getGranter(): smartcore_bos_actor_v1_actor_pb.Actor | undefined;
  setGranter(value?: smartcore_bos_actor_v1_actor_pb.Actor): AccessGrant;
  hasGranter(): boolean;
  clearGranter(): AccessGrant;

  getCreatedTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedTime(value?: google_protobuf_timestamp_pb.Timestamp): AccessGrant;
  hasCreatedTime(): boolean;
  clearCreatedTime(): AccessGrant;

  getUpdatedTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUpdatedTime(value?: google_protobuf_timestamp_pb.Timestamp): AccessGrant;
  hasUpdatedTime(): boolean;
  clearUpdatedTime(): AccessGrant;

  getEntryCode(): string;
  setEntryCode(value: string): AccessGrant;
  hasEntryCode(): boolean;
  clearEntryCode(): AccessGrant;

  getQrCodeUrl(): string;
  setQrCodeUrl(value: string): AccessGrant;
  hasQrCodeUrl(): boolean;
  clearQrCodeUrl(): AccessGrant;

  getQrCodeImage(): Uint8Array | string;
  getQrCodeImage_asU8(): Uint8Array;
  getQrCodeImage_asB64(): string;
  setQrCodeImage(value: Uint8Array | string): AccessGrant;
  hasQrCodeImage(): boolean;
  clearQrCodeImage(): AccessGrant;

  getSkipCheckIn(): boolean;
  setSkipCheckIn(value: boolean): AccessGrant;
  hasSkipCheckIn(): boolean;
  clearSkipCheckIn(): AccessGrant;

  getQrCodeCase(): AccessGrant.QrCodeCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AccessGrant.AsObject;
  static toObject(includeInstance: boolean, msg: AccessGrant): AccessGrant.AsObject;
  static serializeBinaryToWriter(message: AccessGrant, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AccessGrant;
  static deserializeBinaryFromReader(message: AccessGrant, reader: jspb.BinaryReader): AccessGrant;
}

export namespace AccessGrant {
  export type AsObject = {
    id: string;
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    purpose?: string;
    grantee?: smartcore_bos_actor_v1_actor_pb.Actor.AsObject;
    granter?: smartcore_bos_actor_v1_actor_pb.Actor.AsObject;
    createdTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    updatedTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    entryCode?: string;
    qrCodeUrl?: string;
    qrCodeImage?: Uint8Array | string;
    skipCheckIn?: boolean;
  };

  export enum QrCodeCase {
    QR_CODE_NOT_SET = 0,
    QR_CODE_URL = 10,
    QR_CODE_IMAGE = 11,
  }

  export enum PurposeCase {
    _PURPOSE_NOT_SET = 0,
    PURPOSE = 4,
  }

  export enum CreatedTimeCase {
    _CREATED_TIME_NOT_SET = 0,
    CREATED_TIME = 7,
  }

  export enum UpdatedTimeCase {
    _UPDATED_TIME_NOT_SET = 0,
    UPDATED_TIME = 8,
  }

  export enum EntryCodeCase {
    _ENTRY_CODE_NOT_SET = 0,
    ENTRY_CODE = 9,
  }

  export enum SkipCheckInCase {
    _SKIP_CHECK_IN_NOT_SET = 0,
    SKIP_CHECK_IN = 12,
  }
}

export class CreateAccessGrantRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CreateAccessGrantRequest;

  getAccessGrant(): AccessGrant | undefined;
  setAccessGrant(value?: AccessGrant): CreateAccessGrantRequest;
  hasAccessGrant(): boolean;
  clearAccessGrant(): CreateAccessGrantRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateAccessGrantRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateAccessGrantRequest): CreateAccessGrantRequest.AsObject;
  static serializeBinaryToWriter(message: CreateAccessGrantRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateAccessGrantRequest;
  static deserializeBinaryFromReader(message: CreateAccessGrantRequest, reader: jspb.BinaryReader): CreateAccessGrantRequest;
}

export namespace CreateAccessGrantRequest {
  export type AsObject = {
    name: string;
    accessGrant?: AccessGrant.AsObject;
  };
}

export class UpdateAccessGrantRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateAccessGrantRequest;

  getAccessGrant(): AccessGrant | undefined;
  setAccessGrant(value?: AccessGrant): UpdateAccessGrantRequest;
  hasAccessGrant(): boolean;
  clearAccessGrant(): UpdateAccessGrantRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateAccessGrantRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateAccessGrantRequest): UpdateAccessGrantRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateAccessGrantRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateAccessGrantRequest;
  static deserializeBinaryFromReader(message: UpdateAccessGrantRequest, reader: jspb.BinaryReader): UpdateAccessGrantRequest;
}

export namespace UpdateAccessGrantRequest {
  export type AsObject = {
    name: string;
    accessGrant?: AccessGrant.AsObject;
  };
}

export class DeleteAccessGrantRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DeleteAccessGrantRequest;

  getAccessGrantId(): string;
  setAccessGrantId(value: string): DeleteAccessGrantRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteAccessGrantRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteAccessGrantRequest): DeleteAccessGrantRequest.AsObject;
  static serializeBinaryToWriter(message: DeleteAccessGrantRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteAccessGrantRequest;
  static deserializeBinaryFromReader(message: DeleteAccessGrantRequest, reader: jspb.BinaryReader): DeleteAccessGrantRequest;
}

export namespace DeleteAccessGrantRequest {
  export type AsObject = {
    name: string;
    accessGrantId: string;
  };
}

export class DeleteAccessGrantResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteAccessGrantResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteAccessGrantResponse): DeleteAccessGrantResponse.AsObject;
  static serializeBinaryToWriter(message: DeleteAccessGrantResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteAccessGrantResponse;
  static deserializeBinaryFromReader(message: DeleteAccessGrantResponse, reader: jspb.BinaryReader): DeleteAccessGrantResponse;
}

export namespace DeleteAccessGrantResponse {
  export type AsObject = {
  };
}

export class GetAccessGrantsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetAccessGrantsRequest;

  getAccessGrantId(): string;
  setAccessGrantId(value: string): GetAccessGrantsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetAccessGrantsRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetAccessGrantsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetAccessGrantsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetAccessGrantsRequest): GetAccessGrantsRequest.AsObject;
  static serializeBinaryToWriter(message: GetAccessGrantsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetAccessGrantsRequest;
  static deserializeBinaryFromReader(message: GetAccessGrantsRequest, reader: jspb.BinaryReader): GetAccessGrantsRequest;
}

export namespace GetAccessGrantsRequest {
  export type AsObject = {
    name: string;
    accessGrantId: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class ListAccessGrantsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListAccessGrantsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListAccessGrantsRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListAccessGrantsRequest;

  getPageSize(): number;
  setPageSize(value: number): ListAccessGrantsRequest;

  getPageToken(): string;
  setPageToken(value: string): ListAccessGrantsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAccessGrantsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListAccessGrantsRequest): ListAccessGrantsRequest.AsObject;
  static serializeBinaryToWriter(message: ListAccessGrantsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAccessGrantsRequest;
  static deserializeBinaryFromReader(message: ListAccessGrantsRequest, reader: jspb.BinaryReader): ListAccessGrantsRequest;
}

export namespace ListAccessGrantsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
  };
}

export class ListAccessGrantsResponse extends jspb.Message {
  getAccessGrantsList(): Array<AccessGrant>;
  setAccessGrantsList(value: Array<AccessGrant>): ListAccessGrantsResponse;
  clearAccessGrantsList(): ListAccessGrantsResponse;
  addAccessGrants(value?: AccessGrant, index?: number): AccessGrant;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListAccessGrantsResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListAccessGrantsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAccessGrantsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListAccessGrantsResponse): ListAccessGrantsResponse.AsObject;
  static serializeBinaryToWriter(message: ListAccessGrantsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAccessGrantsResponse;
  static deserializeBinaryFromReader(message: ListAccessGrantsResponse, reader: jspb.BinaryReader): ListAccessGrantsResponse;
}

export namespace ListAccessGrantsResponse {
  export type AsObject = {
    accessGrantsList: Array<AccessGrant.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

