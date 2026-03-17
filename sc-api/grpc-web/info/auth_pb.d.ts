import * as jspb from 'google-protobuf'

import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_change_pb from '../types/change_pb'; // proto import: "types/change.proto"


export class Account extends jspb.Message {
  getName(): string;
  setName(value: string): Account;

  getTitle(): string;
  setTitle(value: string): Account;

  getToken(): Token | undefined;
  setToken(value?: Token): Account;
  hasToken(): boolean;
  clearToken(): Account;

  getPermissionsList(): Array<Permission>;
  setPermissionsList(value: Array<Permission>): Account;
  clearPermissionsList(): Account;
  addPermissions(value?: Permission, index?: number): Permission;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Account.AsObject;
  static toObject(includeInstance: boolean, msg: Account): Account.AsObject;
  static serializeBinaryToWriter(message: Account, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Account;
  static deserializeBinaryFromReader(message: Account, reader: jspb.BinaryReader): Account;
}

export namespace Account {
  export type AsObject = {
    name: string;
    title: string;
    token?: Token.AsObject;
    permissionsList: Array<Permission.AsObject>;
  };
}

export class Token extends jspb.Message {
  getId(): string;
  setId(value: string): Token;

  getExpiresAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setExpiresAt(value?: google_protobuf_timestamp_pb.Timestamp): Token;
  hasExpiresAt(): boolean;
  clearExpiresAt(): Token;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Token.AsObject;
  static toObject(includeInstance: boolean, msg: Token): Token.AsObject;
  static serializeBinaryToWriter(message: Token, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Token;
  static deserializeBinaryFromReader(message: Token, reader: jspb.BinaryReader): Token;
}

export namespace Token {
  export type AsObject = {
    id: string;
    expiresAt?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class Permission extends jspb.Message {
  getDeviceName(): string;
  setDeviceName(value: string): Permission;

  getTraitName(): string;
  setTraitName(value: string): Permission;

  getRead(): boolean;
  setRead(value: boolean): Permission;

  getWrite(): boolean;
  setWrite(value: boolean): Permission;

  getObserve(): boolean;
  setObserve(value: boolean): Permission;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Permission.AsObject;
  static toObject(includeInstance: boolean, msg: Permission): Permission.AsObject;
  static serializeBinaryToWriter(message: Permission, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Permission;
  static deserializeBinaryFromReader(message: Permission, reader: jspb.BinaryReader): Permission;
}

export namespace Permission {
  export type AsObject = {
    deviceName: string;
    traitName: string;
    read: boolean;
    write: boolean;
    observe: boolean;
  };
}

export class AddAccountRequest extends jspb.Message {
  getName(): string;
  setName(value: string): AddAccountRequest;

  getTitle(): string;
  setTitle(value: string): AddAccountRequest;

  getPermissionsList(): Array<Permission>;
  setPermissionsList(value: Array<Permission>): AddAccountRequest;
  clearPermissionsList(): AddAccountRequest;
  addPermissions(value?: Permission, index?: number): Permission;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AddAccountRequest.AsObject;
  static toObject(includeInstance: boolean, msg: AddAccountRequest): AddAccountRequest.AsObject;
  static serializeBinaryToWriter(message: AddAccountRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AddAccountRequest;
  static deserializeBinaryFromReader(message: AddAccountRequest, reader: jspb.BinaryReader): AddAccountRequest;
}

export namespace AddAccountRequest {
  export type AsObject = {
    name: string;
    title: string;
    permissionsList: Array<Permission.AsObject>;
  };
}

export class AddAccountResponse extends jspb.Message {
  getAccount(): Account | undefined;
  setAccount(value?: Account): AddAccountResponse;
  hasAccount(): boolean;
  clearAccount(): AddAccountResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AddAccountResponse.AsObject;
  static toObject(includeInstance: boolean, msg: AddAccountResponse): AddAccountResponse.AsObject;
  static serializeBinaryToWriter(message: AddAccountResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AddAccountResponse;
  static deserializeBinaryFromReader(message: AddAccountResponse, reader: jspb.BinaryReader): AddAccountResponse;
}

export namespace AddAccountResponse {
  export type AsObject = {
    account?: Account.AsObject;
  };
}

export class RemoveAccountRequest extends jspb.Message {
  getName(): string;
  setName(value: string): RemoveAccountRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RemoveAccountRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RemoveAccountRequest): RemoveAccountRequest.AsObject;
  static serializeBinaryToWriter(message: RemoveAccountRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RemoveAccountRequest;
  static deserializeBinaryFromReader(message: RemoveAccountRequest, reader: jspb.BinaryReader): RemoveAccountRequest;
}

export namespace RemoveAccountRequest {
  export type AsObject = {
    name: string;
  };
}

export class RemoveAccountResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RemoveAccountResponse.AsObject;
  static toObject(includeInstance: boolean, msg: RemoveAccountResponse): RemoveAccountResponse.AsObject;
  static serializeBinaryToWriter(message: RemoveAccountResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RemoveAccountResponse;
  static deserializeBinaryFromReader(message: RemoveAccountResponse, reader: jspb.BinaryReader): RemoveAccountResponse;
}

export namespace RemoveAccountResponse {
  export type AsObject = {
  };
}

export class UpdateAccountPermissionsRequest extends jspb.Message {
  getChangeType(): types_change_pb.ChangeType;
  setChangeType(value: types_change_pb.ChangeType): UpdateAccountPermissionsRequest;

  getPermissionsList(): Array<Permission>;
  setPermissionsList(value: Array<Permission>): UpdateAccountPermissionsRequest;
  clearPermissionsList(): UpdateAccountPermissionsRequest;
  addPermissions(value?: Permission, index?: number): Permission;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateAccountPermissionsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateAccountPermissionsRequest): UpdateAccountPermissionsRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateAccountPermissionsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateAccountPermissionsRequest;
  static deserializeBinaryFromReader(message: UpdateAccountPermissionsRequest, reader: jspb.BinaryReader): UpdateAccountPermissionsRequest;
}

export namespace UpdateAccountPermissionsRequest {
  export type AsObject = {
    changeType: types_change_pb.ChangeType;
    permissionsList: Array<Permission.AsObject>;
  };
}

export class UpdateAccountPermissionsResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateAccountPermissionsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateAccountPermissionsResponse): UpdateAccountPermissionsResponse.AsObject;
  static serializeBinaryToWriter(message: UpdateAccountPermissionsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateAccountPermissionsResponse;
  static deserializeBinaryFromReader(message: UpdateAccountPermissionsResponse, reader: jspb.BinaryReader): UpdateAccountPermissionsResponse;
}

export namespace UpdateAccountPermissionsResponse {
  export type AsObject = {
  };
}

export class GenerateTokenRequest extends jspb.Message {
  getAccountName(): string;
  setAccountName(value: string): GenerateTokenRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GenerateTokenRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GenerateTokenRequest): GenerateTokenRequest.AsObject;
  static serializeBinaryToWriter(message: GenerateTokenRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GenerateTokenRequest;
  static deserializeBinaryFromReader(message: GenerateTokenRequest, reader: jspb.BinaryReader): GenerateTokenRequest;
}

export namespace GenerateTokenRequest {
  export type AsObject = {
    accountName: string;
  };
}

export class GenerateTokenResponse extends jspb.Message {
  getToken(): Token | undefined;
  setToken(value?: Token): GenerateTokenResponse;
  hasToken(): boolean;
  clearToken(): GenerateTokenResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GenerateTokenResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GenerateTokenResponse): GenerateTokenResponse.AsObject;
  static serializeBinaryToWriter(message: GenerateTokenResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GenerateTokenResponse;
  static deserializeBinaryFromReader(message: GenerateTokenResponse, reader: jspb.BinaryReader): GenerateTokenResponse;
}

export namespace GenerateTokenResponse {
  export type AsObject = {
    token?: Token.AsObject;
  };
}

