import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_types_v1_info_pb from '../../../../smartcore/bos/types/v1/info_pb'; // proto import: "smartcore/bos/types/v1/info.proto"


export class Credential extends jspb.Message {
  getId(): string;
  setId(value: string): Credential;

  getType(): string;
  setType(value: string): Credential;

  getKind(): Credential.Kind;
  setKind(value: Credential.Kind): Credential;

  getValue(): string;
  setValue(value: string): Credential;

  getState(): Credential.State;
  setState(value: Credential.State): Credential;

  getNativeState(): string;
  setNativeState(value: string): Credential;

  getActiveTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setActiveTime(value?: google_protobuf_timestamp_pb.Timestamp): Credential;
  hasActiveTime(): boolean;
  clearActiveTime(): Credential;

  getExpireTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setExpireTime(value?: google_protobuf_timestamp_pb.Timestamp): Credential;
  hasExpireTime(): boolean;
  clearExpireTime(): Credential;

  getIssueLevel(): number;
  setIssueLevel(value: number): Credential;
  hasIssueLevel(): boolean;
  clearIssueLevel(): Credential;

  getInvitation(): Credential.Invitation | undefined;
  setInvitation(value?: Credential.Invitation): Credential;
  hasInvitation(): boolean;
  clearInvitation(): Credential;

  getMoreMap(): jspb.Map<string, string>;
  clearMoreMap(): Credential;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Credential.AsObject;
  static toObject(includeInstance: boolean, msg: Credential): Credential.AsObject;
  static serializeBinaryToWriter(message: Credential, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Credential;
  static deserializeBinaryFromReader(message: Credential, reader: jspb.BinaryReader): Credential;
}

export namespace Credential {
  export type AsObject = {
    id: string;
    type: string;
    kind: Credential.Kind;
    value: string;
    state: Credential.State;
    nativeState: string;
    activeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    expireTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    issueLevel?: number;
    invitation?: Credential.Invitation.AsObject;
    moreMap: Array<[string, string]>;
  };

  export class Invitation extends jspb.Message {
    getEmail(): string;
    setEmail(value: string): Invitation;

    getPhoneNumber(): string;
    setPhoneNumber(value: string): Invitation;

    getStatus(): string;
    setStatus(value: string): Invitation;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Invitation.AsObject;
    static toObject(includeInstance: boolean, msg: Invitation): Invitation.AsObject;
    static serializeBinaryToWriter(message: Invitation, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Invitation;
    static deserializeBinaryFromReader(message: Invitation, reader: jspb.BinaryReader): Invitation;
  }

  export namespace Invitation {
    export type AsObject = {
      email: string;
      phoneNumber: string;
      status: string;
    };
  }


  export enum Kind {
    KIND_UNSPECIFIED = 0,
    CARD = 1,
    FOB = 2,
    MOBILE = 3,
    VEHICLE_PLATE = 4,
    PHONE_NUMBER = 5,
    BIOMETRIC = 6,
    PIN = 7,
  }

  export enum State {
    STATE_UNSPECIFIED = 0,
    ACTIVE = 1,
    DISABLED = 2,
    LOST = 3,
    STOLEN = 4,
    EXPIRED = 5,
    PENDING = 6,
  }

  export enum IssueLevelCase {
    _ISSUE_LEVEL_NOT_SET = 0,
    ISSUE_LEVEL = 9,
  }
}

export class GetCredentialRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetCredentialRequest;

  getId(): string;
  setId(value: string): GetCredentialRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetCredentialRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetCredentialRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetCredentialRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetCredentialRequest): GetCredentialRequest.AsObject;
  static serializeBinaryToWriter(message: GetCredentialRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetCredentialRequest;
  static deserializeBinaryFromReader(message: GetCredentialRequest, reader: jspb.BinaryReader): GetCredentialRequest;
}

export namespace GetCredentialRequest {
  export type AsObject = {
    name: string;
    id: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class ListCredentialsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListCredentialsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListCredentialsRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListCredentialsRequest;

  getPageSize(): number;
  setPageSize(value: number): ListCredentialsRequest;

  getPageToken(): string;
  setPageToken(value: string): ListCredentialsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListCredentialsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListCredentialsRequest): ListCredentialsRequest.AsObject;
  static serializeBinaryToWriter(message: ListCredentialsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListCredentialsRequest;
  static deserializeBinaryFromReader(message: ListCredentialsRequest, reader: jspb.BinaryReader): ListCredentialsRequest;
}

export namespace ListCredentialsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
  };
}

export class ListCredentialsResponse extends jspb.Message {
  getCredentialsList(): Array<Credential>;
  setCredentialsList(value: Array<Credential>): ListCredentialsResponse;
  clearCredentialsList(): ListCredentialsResponse;
  addCredentials(value?: Credential, index?: number): Credential;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListCredentialsResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListCredentialsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListCredentialsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListCredentialsResponse): ListCredentialsResponse.AsObject;
  static serializeBinaryToWriter(message: ListCredentialsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListCredentialsResponse;
  static deserializeBinaryFromReader(message: ListCredentialsResponse, reader: jspb.BinaryReader): ListCredentialsResponse;
}

export namespace ListCredentialsResponse {
  export type AsObject = {
    credentialsList: Array<Credential.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class CreateCredentialRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CreateCredentialRequest;

  getCredential(): Credential | undefined;
  setCredential(value?: Credential): CreateCredentialRequest;
  hasCredential(): boolean;
  clearCredential(): CreateCredentialRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateCredentialRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateCredentialRequest): CreateCredentialRequest.AsObject;
  static serializeBinaryToWriter(message: CreateCredentialRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateCredentialRequest;
  static deserializeBinaryFromReader(message: CreateCredentialRequest, reader: jspb.BinaryReader): CreateCredentialRequest;
}

export namespace CreateCredentialRequest {
  export type AsObject = {
    name: string;
    credential?: Credential.AsObject;
  };
}

export class UpdateCredentialRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateCredentialRequest;

  getCredential(): Credential | undefined;
  setCredential(value?: Credential): UpdateCredentialRequest;
  hasCredential(): boolean;
  clearCredential(): UpdateCredentialRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateCredentialRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateCredentialRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateCredentialRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateCredentialRequest): UpdateCredentialRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateCredentialRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateCredentialRequest;
  static deserializeBinaryFromReader(message: UpdateCredentialRequest, reader: jspb.BinaryReader): UpdateCredentialRequest;
}

export namespace UpdateCredentialRequest {
  export type AsObject = {
    name: string;
    credential?: Credential.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class DeleteCredentialRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DeleteCredentialRequest;

  getId(): string;
  setId(value: string): DeleteCredentialRequest;

  getAllowMissing(): boolean;
  setAllowMissing(value: boolean): DeleteCredentialRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteCredentialRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteCredentialRequest): DeleteCredentialRequest.AsObject;
  static serializeBinaryToWriter(message: DeleteCredentialRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteCredentialRequest;
  static deserializeBinaryFromReader(message: DeleteCredentialRequest, reader: jspb.BinaryReader): DeleteCredentialRequest;
}

export namespace DeleteCredentialRequest {
  export type AsObject = {
    name: string;
    id: string;
    allowMissing: boolean;
  };
}

export class DeleteCredentialResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteCredentialResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteCredentialResponse): DeleteCredentialResponse.AsObject;
  static serializeBinaryToWriter(message: DeleteCredentialResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteCredentialResponse;
  static deserializeBinaryFromReader(message: DeleteCredentialResponse, reader: jspb.BinaryReader): DeleteCredentialResponse;
}

export namespace DeleteCredentialResponse {
  export type AsObject = {
  };
}

export class DescribeCredentialRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeCredentialRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeCredentialRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeCredentialRequest): DescribeCredentialRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeCredentialRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeCredentialRequest;
  static deserializeBinaryFromReader(message: DescribeCredentialRequest, reader: jspb.BinaryReader): DescribeCredentialRequest;
}

export namespace DescribeCredentialRequest {
  export type AsObject = {
    name: string;
  };
}

export class CredentialSupport extends jspb.Message {
  getResourceSupport(): smartcore_bos_types_v1_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: smartcore_bos_types_v1_info_pb.ResourceSupport): CredentialSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): CredentialSupport;

  getTypesList(): Array<CredentialType>;
  setTypesList(value: Array<CredentialType>): CredentialSupport;
  clearTypesList(): CredentialSupport;
  addTypes(value?: CredentialType, index?: number): CredentialType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CredentialSupport.AsObject;
  static toObject(includeInstance: boolean, msg: CredentialSupport): CredentialSupport.AsObject;
  static serializeBinaryToWriter(message: CredentialSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CredentialSupport;
  static deserializeBinaryFromReader(message: CredentialSupport, reader: jspb.BinaryReader): CredentialSupport;
}

export namespace CredentialSupport {
  export type AsObject = {
    resourceSupport?: smartcore_bos_types_v1_info_pb.ResourceSupport.AsObject;
    typesList: Array<CredentialType.AsObject>;
  };
}

export class CredentialType extends jspb.Message {
  getId(): string;
  setId(value: string): CredentialType;

  getDisplayName(): string;
  setDisplayName(value: string): CredentialType;

  getKind(): Credential.Kind;
  setKind(value: Credential.Kind): CredentialType;

  getValueSource(): CredentialType.ValueSource;
  setValueSource(value: CredentialType.ValueSource): CredentialType;

  getWritableStatesList(): Array<Credential.State>;
  setWritableStatesList(value: Array<Credential.State>): CredentialType;
  clearWritableStatesList(): CredentialType;
  addWritableStates(value: Credential.State, index?: number): CredentialType;

  getNativeStatesList(): Array<CredentialType.NativeState>;
  setNativeStatesList(value: Array<CredentialType.NativeState>): CredentialType;
  clearNativeStatesList(): CredentialType;
  addNativeStates(value?: CredentialType.NativeState, index?: number): CredentialType.NativeState;

  getInvitationRequired(): boolean;
  setInvitationRequired(value: boolean): CredentialType;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CredentialType.AsObject;
  static toObject(includeInstance: boolean, msg: CredentialType): CredentialType.AsObject;
  static serializeBinaryToWriter(message: CredentialType, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CredentialType;
  static deserializeBinaryFromReader(message: CredentialType, reader: jspb.BinaryReader): CredentialType;
}

export namespace CredentialType {
  export type AsObject = {
    id: string;
    displayName: string;
    kind: Credential.Kind;
    valueSource: CredentialType.ValueSource;
    writableStatesList: Array<Credential.State>;
    nativeStatesList: Array<CredentialType.NativeState.AsObject>;
    invitationRequired: boolean;
  };

  export class NativeState extends jspb.Message {
    getId(): string;
    setId(value: string): NativeState;

    getDisplayName(): string;
    setDisplayName(value: string): NativeState;

    getState(): Credential.State;
    setState(value: Credential.State): NativeState;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): NativeState.AsObject;
    static toObject(includeInstance: boolean, msg: NativeState): NativeState.AsObject;
    static serializeBinaryToWriter(message: NativeState, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): NativeState;
    static deserializeBinaryFromReader(message: NativeState, reader: jspb.BinaryReader): NativeState;
  }

  export namespace NativeState {
    export type AsObject = {
      id: string;
      displayName: string;
      state: Credential.State;
    };
  }


  export enum ValueSource {
    VALUE_SOURCE_UNSPECIFIED = 0,
    CALLER_SUPPLIED = 1,
    SYSTEM_ALLOCATED = 2,
    EITHER = 3,
  }
}

