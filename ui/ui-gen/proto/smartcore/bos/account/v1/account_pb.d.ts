import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class GetAccountRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetAccountRequest;

  getId(): string;
  setId(value: string): GetAccountRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetAccountRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetAccountRequest): GetAccountRequest.AsObject;
  static serializeBinaryToWriter(message: GetAccountRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetAccountRequest;
  static deserializeBinaryFromReader(message: GetAccountRequest, reader: jspb.BinaryReader): GetAccountRequest;
}

export namespace GetAccountRequest {
  export type AsObject = {
    name: string;
    id: string;
  };
}

export class CreateAccountRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CreateAccountRequest;

  getAccount(): Account | undefined;
  setAccount(value?: Account): CreateAccountRequest;
  hasAccount(): boolean;
  clearAccount(): CreateAccountRequest;

  getPassword(): string;
  setPassword(value: string): CreateAccountRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateAccountRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateAccountRequest): CreateAccountRequest.AsObject;
  static serializeBinaryToWriter(message: CreateAccountRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateAccountRequest;
  static deserializeBinaryFromReader(message: CreateAccountRequest, reader: jspb.BinaryReader): CreateAccountRequest;
}

export namespace CreateAccountRequest {
  export type AsObject = {
    name: string;
    account?: Account.AsObject;
    password: string;
  };
}

export class ListAccountsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListAccountsRequest;

  getPageSize(): number;
  setPageSize(value: number): ListAccountsRequest;

  getPageToken(): string;
  setPageToken(value: string): ListAccountsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAccountsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListAccountsRequest): ListAccountsRequest.AsObject;
  static serializeBinaryToWriter(message: ListAccountsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAccountsRequest;
  static deserializeBinaryFromReader(message: ListAccountsRequest, reader: jspb.BinaryReader): ListAccountsRequest;
}

export namespace ListAccountsRequest {
  export type AsObject = {
    name: string;
    pageSize: number;
    pageToken: string;
  };
}

export class ListAccountsResponse extends jspb.Message {
  getAccountsList(): Array<Account>;
  setAccountsList(value: Array<Account>): ListAccountsResponse;
  clearAccountsList(): ListAccountsResponse;
  addAccounts(value?: Account, index?: number): Account;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListAccountsResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListAccountsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAccountsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListAccountsResponse): ListAccountsResponse.AsObject;
  static serializeBinaryToWriter(message: ListAccountsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAccountsResponse;
  static deserializeBinaryFromReader(message: ListAccountsResponse, reader: jspb.BinaryReader): ListAccountsResponse;
}

export namespace ListAccountsResponse {
  export type AsObject = {
    accountsList: Array<Account.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class UpdateAccountRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateAccountRequest;

  getAccount(): Account | undefined;
  setAccount(value?: Account): UpdateAccountRequest;
  hasAccount(): boolean;
  clearAccount(): UpdateAccountRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateAccountRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateAccountRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateAccountRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateAccountRequest): UpdateAccountRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateAccountRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateAccountRequest;
  static deserializeBinaryFromReader(message: UpdateAccountRequest, reader: jspb.BinaryReader): UpdateAccountRequest;
}

export namespace UpdateAccountRequest {
  export type AsObject = {
    name: string;
    account?: Account.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateAccountPasswordRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateAccountPasswordRequest;

  getId(): string;
  setId(value: string): UpdateAccountPasswordRequest;

  getNewPassword(): string;
  setNewPassword(value: string): UpdateAccountPasswordRequest;

  getOldPassword(): string;
  setOldPassword(value: string): UpdateAccountPasswordRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateAccountPasswordRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateAccountPasswordRequest): UpdateAccountPasswordRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateAccountPasswordRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateAccountPasswordRequest;
  static deserializeBinaryFromReader(message: UpdateAccountPasswordRequest, reader: jspb.BinaryReader): UpdateAccountPasswordRequest;
}

export namespace UpdateAccountPasswordRequest {
  export type AsObject = {
    name: string;
    id: string;
    newPassword: string;
    oldPassword: string;
  };
}

export class UpdateAccountPasswordResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateAccountPasswordResponse.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateAccountPasswordResponse): UpdateAccountPasswordResponse.AsObject;
  static serializeBinaryToWriter(message: UpdateAccountPasswordResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateAccountPasswordResponse;
  static deserializeBinaryFromReader(message: UpdateAccountPasswordResponse, reader: jspb.BinaryReader): UpdateAccountPasswordResponse;
}

export namespace UpdateAccountPasswordResponse {
  export type AsObject = {
  };
}

export class RotateAccountClientSecretRequest extends jspb.Message {
  getName(): string;
  setName(value: string): RotateAccountClientSecretRequest;

  getId(): string;
  setId(value: string): RotateAccountClientSecretRequest;

  getPreviousSecretExpireTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setPreviousSecretExpireTime(value?: google_protobuf_timestamp_pb.Timestamp): RotateAccountClientSecretRequest;
  hasPreviousSecretExpireTime(): boolean;
  clearPreviousSecretExpireTime(): RotateAccountClientSecretRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RotateAccountClientSecretRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RotateAccountClientSecretRequest): RotateAccountClientSecretRequest.AsObject;
  static serializeBinaryToWriter(message: RotateAccountClientSecretRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RotateAccountClientSecretRequest;
  static deserializeBinaryFromReader(message: RotateAccountClientSecretRequest, reader: jspb.BinaryReader): RotateAccountClientSecretRequest;
}

export namespace RotateAccountClientSecretRequest {
  export type AsObject = {
    name: string;
    id: string;
    previousSecretExpireTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class RotateAccountClientSecretResponse extends jspb.Message {
  getClientSecret(): string;
  setClientSecret(value: string): RotateAccountClientSecretResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RotateAccountClientSecretResponse.AsObject;
  static toObject(includeInstance: boolean, msg: RotateAccountClientSecretResponse): RotateAccountClientSecretResponse.AsObject;
  static serializeBinaryToWriter(message: RotateAccountClientSecretResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RotateAccountClientSecretResponse;
  static deserializeBinaryFromReader(message: RotateAccountClientSecretResponse, reader: jspb.BinaryReader): RotateAccountClientSecretResponse;
}

export namespace RotateAccountClientSecretResponse {
  export type AsObject = {
    clientSecret: string;
  };
}

export class DeleteAccountRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DeleteAccountRequest;

  getId(): string;
  setId(value: string): DeleteAccountRequest;

  getAllowMissing(): boolean;
  setAllowMissing(value: boolean): DeleteAccountRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteAccountRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteAccountRequest): DeleteAccountRequest.AsObject;
  static serializeBinaryToWriter(message: DeleteAccountRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteAccountRequest;
  static deserializeBinaryFromReader(message: DeleteAccountRequest, reader: jspb.BinaryReader): DeleteAccountRequest;
}

export namespace DeleteAccountRequest {
  export type AsObject = {
    name: string;
    id: string;
    allowMissing: boolean;
  };
}

export class DeleteAccountResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteAccountResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteAccountResponse): DeleteAccountResponse.AsObject;
  static serializeBinaryToWriter(message: DeleteAccountResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteAccountResponse;
  static deserializeBinaryFromReader(message: DeleteAccountResponse, reader: jspb.BinaryReader): DeleteAccountResponse;
}

export namespace DeleteAccountResponse {
  export type AsObject = {
  };
}

export class GetRoleRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetRoleRequest;

  getId(): string;
  setId(value: string): GetRoleRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetRoleRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetRoleRequest): GetRoleRequest.AsObject;
  static serializeBinaryToWriter(message: GetRoleRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetRoleRequest;
  static deserializeBinaryFromReader(message: GetRoleRequest, reader: jspb.BinaryReader): GetRoleRequest;
}

export namespace GetRoleRequest {
  export type AsObject = {
    name: string;
    id: string;
  };
}

export class ListRolesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListRolesRequest;

  getPageSize(): number;
  setPageSize(value: number): ListRolesRequest;

  getPageToken(): string;
  setPageToken(value: string): ListRolesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListRolesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListRolesRequest): ListRolesRequest.AsObject;
  static serializeBinaryToWriter(message: ListRolesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListRolesRequest;
  static deserializeBinaryFromReader(message: ListRolesRequest, reader: jspb.BinaryReader): ListRolesRequest;
}

export namespace ListRolesRequest {
  export type AsObject = {
    name: string;
    pageSize: number;
    pageToken: string;
  };
}

export class ListRolesResponse extends jspb.Message {
  getRolesList(): Array<Role>;
  setRolesList(value: Array<Role>): ListRolesResponse;
  clearRolesList(): ListRolesResponse;
  addRoles(value?: Role, index?: number): Role;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListRolesResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListRolesResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListRolesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListRolesResponse): ListRolesResponse.AsObject;
  static serializeBinaryToWriter(message: ListRolesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListRolesResponse;
  static deserializeBinaryFromReader(message: ListRolesResponse, reader: jspb.BinaryReader): ListRolesResponse;
}

export namespace ListRolesResponse {
  export type AsObject = {
    rolesList: Array<Role.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class CreateRoleRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CreateRoleRequest;

  getRole(): Role | undefined;
  setRole(value?: Role): CreateRoleRequest;
  hasRole(): boolean;
  clearRole(): CreateRoleRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateRoleRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateRoleRequest): CreateRoleRequest.AsObject;
  static serializeBinaryToWriter(message: CreateRoleRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateRoleRequest;
  static deserializeBinaryFromReader(message: CreateRoleRequest, reader: jspb.BinaryReader): CreateRoleRequest;
}

export namespace CreateRoleRequest {
  export type AsObject = {
    name: string;
    role?: Role.AsObject;
  };
}

export class UpdateRoleRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateRoleRequest;

  getRole(): Role | undefined;
  setRole(value?: Role): UpdateRoleRequest;
  hasRole(): boolean;
  clearRole(): UpdateRoleRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateRoleRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateRoleRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateRoleRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateRoleRequest): UpdateRoleRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateRoleRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateRoleRequest;
  static deserializeBinaryFromReader(message: UpdateRoleRequest, reader: jspb.BinaryReader): UpdateRoleRequest;
}

export namespace UpdateRoleRequest {
  export type AsObject = {
    name: string;
    role?: Role.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class DeleteRoleRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DeleteRoleRequest;

  getId(): string;
  setId(value: string): DeleteRoleRequest;

  getAllowMissing(): boolean;
  setAllowMissing(value: boolean): DeleteRoleRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteRoleRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteRoleRequest): DeleteRoleRequest.AsObject;
  static serializeBinaryToWriter(message: DeleteRoleRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteRoleRequest;
  static deserializeBinaryFromReader(message: DeleteRoleRequest, reader: jspb.BinaryReader): DeleteRoleRequest;
}

export namespace DeleteRoleRequest {
  export type AsObject = {
    name: string;
    id: string;
    allowMissing: boolean;
  };
}

export class DeleteRoleResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteRoleResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteRoleResponse): DeleteRoleResponse.AsObject;
  static serializeBinaryToWriter(message: DeleteRoleResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteRoleResponse;
  static deserializeBinaryFromReader(message: DeleteRoleResponse, reader: jspb.BinaryReader): DeleteRoleResponse;
}

export namespace DeleteRoleResponse {
  export type AsObject = {
  };
}

export class GetRoleAssignmentRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetRoleAssignmentRequest;

  getId(): string;
  setId(value: string): GetRoleAssignmentRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetRoleAssignmentRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetRoleAssignmentRequest): GetRoleAssignmentRequest.AsObject;
  static serializeBinaryToWriter(message: GetRoleAssignmentRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetRoleAssignmentRequest;
  static deserializeBinaryFromReader(message: GetRoleAssignmentRequest, reader: jspb.BinaryReader): GetRoleAssignmentRequest;
}

export namespace GetRoleAssignmentRequest {
  export type AsObject = {
    name: string;
    id: string;
  };
}

export class ListRoleAssignmentsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListRoleAssignmentsRequest;

  getPageSize(): number;
  setPageSize(value: number): ListRoleAssignmentsRequest;

  getPageToken(): string;
  setPageToken(value: string): ListRoleAssignmentsRequest;

  getFilter(): string;
  setFilter(value: string): ListRoleAssignmentsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListRoleAssignmentsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListRoleAssignmentsRequest): ListRoleAssignmentsRequest.AsObject;
  static serializeBinaryToWriter(message: ListRoleAssignmentsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListRoleAssignmentsRequest;
  static deserializeBinaryFromReader(message: ListRoleAssignmentsRequest, reader: jspb.BinaryReader): ListRoleAssignmentsRequest;
}

export namespace ListRoleAssignmentsRequest {
  export type AsObject = {
    name: string;
    pageSize: number;
    pageToken: string;
    filter: string;
  };
}

export class ListRoleAssignmentsResponse extends jspb.Message {
  getRoleAssignmentsList(): Array<RoleAssignment>;
  setRoleAssignmentsList(value: Array<RoleAssignment>): ListRoleAssignmentsResponse;
  clearRoleAssignmentsList(): ListRoleAssignmentsResponse;
  addRoleAssignments(value?: RoleAssignment, index?: number): RoleAssignment;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListRoleAssignmentsResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListRoleAssignmentsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListRoleAssignmentsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListRoleAssignmentsResponse): ListRoleAssignmentsResponse.AsObject;
  static serializeBinaryToWriter(message: ListRoleAssignmentsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListRoleAssignmentsResponse;
  static deserializeBinaryFromReader(message: ListRoleAssignmentsResponse, reader: jspb.BinaryReader): ListRoleAssignmentsResponse;
}

export namespace ListRoleAssignmentsResponse {
  export type AsObject = {
    roleAssignmentsList: Array<RoleAssignment.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class CreateRoleAssignmentRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CreateRoleAssignmentRequest;

  getRoleAssignment(): RoleAssignment | undefined;
  setRoleAssignment(value?: RoleAssignment): CreateRoleAssignmentRequest;
  hasRoleAssignment(): boolean;
  clearRoleAssignment(): CreateRoleAssignmentRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateRoleAssignmentRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateRoleAssignmentRequest): CreateRoleAssignmentRequest.AsObject;
  static serializeBinaryToWriter(message: CreateRoleAssignmentRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateRoleAssignmentRequest;
  static deserializeBinaryFromReader(message: CreateRoleAssignmentRequest, reader: jspb.BinaryReader): CreateRoleAssignmentRequest;
}

export namespace CreateRoleAssignmentRequest {
  export type AsObject = {
    name: string;
    roleAssignment?: RoleAssignment.AsObject;
  };
}

export class DeleteRoleAssignmentRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DeleteRoleAssignmentRequest;

  getId(): string;
  setId(value: string): DeleteRoleAssignmentRequest;

  getAllowMissing(): boolean;
  setAllowMissing(value: boolean): DeleteRoleAssignmentRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteRoleAssignmentRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteRoleAssignmentRequest): DeleteRoleAssignmentRequest.AsObject;
  static serializeBinaryToWriter(message: DeleteRoleAssignmentRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteRoleAssignmentRequest;
  static deserializeBinaryFromReader(message: DeleteRoleAssignmentRequest, reader: jspb.BinaryReader): DeleteRoleAssignmentRequest;
}

export namespace DeleteRoleAssignmentRequest {
  export type AsObject = {
    name: string;
    id: string;
    allowMissing: boolean;
  };
}

export class DeleteRoleAssignmentResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteRoleAssignmentResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteRoleAssignmentResponse): DeleteRoleAssignmentResponse.AsObject;
  static serializeBinaryToWriter(message: DeleteRoleAssignmentResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteRoleAssignmentResponse;
  static deserializeBinaryFromReader(message: DeleteRoleAssignmentResponse, reader: jspb.BinaryReader): DeleteRoleAssignmentResponse;
}

export namespace DeleteRoleAssignmentResponse {
  export type AsObject = {
  };
}

export class GetPermissionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetPermissionRequest;

  getId(): string;
  setId(value: string): GetPermissionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetPermissionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetPermissionRequest): GetPermissionRequest.AsObject;
  static serializeBinaryToWriter(message: GetPermissionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetPermissionRequest;
  static deserializeBinaryFromReader(message: GetPermissionRequest, reader: jspb.BinaryReader): GetPermissionRequest;
}

export namespace GetPermissionRequest {
  export type AsObject = {
    name: string;
    id: string;
  };
}

export class ListPermissionsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListPermissionsRequest;

  getPageSize(): number;
  setPageSize(value: number): ListPermissionsRequest;

  getPageToken(): string;
  setPageToken(value: string): ListPermissionsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListPermissionsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListPermissionsRequest): ListPermissionsRequest.AsObject;
  static serializeBinaryToWriter(message: ListPermissionsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListPermissionsRequest;
  static deserializeBinaryFromReader(message: ListPermissionsRequest, reader: jspb.BinaryReader): ListPermissionsRequest;
}

export namespace ListPermissionsRequest {
  export type AsObject = {
    name: string;
    pageSize: number;
    pageToken: string;
  };
}

export class ListPermissionsResponse extends jspb.Message {
  getPermissionsList(): Array<Permission>;
  setPermissionsList(value: Array<Permission>): ListPermissionsResponse;
  clearPermissionsList(): ListPermissionsResponse;
  addPermissions(value?: Permission, index?: number): Permission;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListPermissionsResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListPermissionsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListPermissionsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListPermissionsResponse): ListPermissionsResponse.AsObject;
  static serializeBinaryToWriter(message: ListPermissionsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListPermissionsResponse;
  static deserializeBinaryFromReader(message: ListPermissionsResponse, reader: jspb.BinaryReader): ListPermissionsResponse;
}

export namespace ListPermissionsResponse {
  export type AsObject = {
    permissionsList: Array<Permission.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class GetAccountLimitsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetAccountLimitsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetAccountLimitsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetAccountLimitsRequest): GetAccountLimitsRequest.AsObject;
  static serializeBinaryToWriter(message: GetAccountLimitsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetAccountLimitsRequest;
  static deserializeBinaryFromReader(message: GetAccountLimitsRequest, reader: jspb.BinaryReader): GetAccountLimitsRequest;
}

export namespace GetAccountLimitsRequest {
  export type AsObject = {
    name: string;
  };
}

export class Account extends jspb.Message {
  getId(): string;
  setId(value: string): Account;

  getCreateTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreateTime(value?: google_protobuf_timestamp_pb.Timestamp): Account;
  hasCreateTime(): boolean;
  clearCreateTime(): Account;

  getType(): Account.Type;
  setType(value: Account.Type): Account;

  getDisplayName(): string;
  setDisplayName(value: string): Account;

  getDescription(): string;
  setDescription(value: string): Account;

  getUserDetails(): UserAccount | undefined;
  setUserDetails(value?: UserAccount): Account;
  hasUserDetails(): boolean;
  clearUserDetails(): Account;

  getServiceDetails(): ServiceAccount | undefined;
  setServiceDetails(value?: ServiceAccount): Account;
  hasServiceDetails(): boolean;
  clearServiceDetails(): Account;

  getDetailsCase(): Account.DetailsCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Account.AsObject;
  static toObject(includeInstance: boolean, msg: Account): Account.AsObject;
  static serializeBinaryToWriter(message: Account, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Account;
  static deserializeBinaryFromReader(message: Account, reader: jspb.BinaryReader): Account;
}

export namespace Account {
  export type AsObject = {
    id: string;
    createTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    type: Account.Type;
    displayName: string;
    description: string;
    userDetails?: UserAccount.AsObject;
    serviceDetails?: ServiceAccount.AsObject;
  };

  export enum Type {
    ACCOUNT_TYPE_UNSPECIFIED = 0,
    USER_ACCOUNT = 1,
    SERVICE_ACCOUNT = 2,
  }

  export enum DetailsCase {
    DETAILS_NOT_SET = 0,
    USER_DETAILS = 6,
    SERVICE_DETAILS = 7,
  }
}

export class UserAccount extends jspb.Message {
  getUsername(): string;
  setUsername(value: string): UserAccount;

  getHasPassword(): boolean;
  setHasPassword(value: boolean): UserAccount;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UserAccount.AsObject;
  static toObject(includeInstance: boolean, msg: UserAccount): UserAccount.AsObject;
  static serializeBinaryToWriter(message: UserAccount, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UserAccount;
  static deserializeBinaryFromReader(message: UserAccount, reader: jspb.BinaryReader): UserAccount;
}

export namespace UserAccount {
  export type AsObject = {
    username: string;
    hasPassword: boolean;
  };
}

export class ServiceAccount extends jspb.Message {
  getClientId(): string;
  setClientId(value: string): ServiceAccount;

  getClientSecret(): string;
  setClientSecret(value: string): ServiceAccount;

  getPreviousSecretExpireTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setPreviousSecretExpireTime(value?: google_protobuf_timestamp_pb.Timestamp): ServiceAccount;
  hasPreviousSecretExpireTime(): boolean;
  clearPreviousSecretExpireTime(): ServiceAccount;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ServiceAccount.AsObject;
  static toObject(includeInstance: boolean, msg: ServiceAccount): ServiceAccount.AsObject;
  static serializeBinaryToWriter(message: ServiceAccount, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ServiceAccount;
  static deserializeBinaryFromReader(message: ServiceAccount, reader: jspb.BinaryReader): ServiceAccount;
}

export namespace ServiceAccount {
  export type AsObject = {
    clientId: string;
    clientSecret: string;
    previousSecretExpireTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class Role extends jspb.Message {
  getId(): string;
  setId(value: string): Role;

  getDisplayName(): string;
  setDisplayName(value: string): Role;

  getDescription(): string;
  setDescription(value: string): Role;

  getPermissionIdsList(): Array<string>;
  setPermissionIdsList(value: Array<string>): Role;
  clearPermissionIdsList(): Role;
  addPermissionIds(value: string, index?: number): Role;

  getLegacyRoleName(): string;
  setLegacyRoleName(value: string): Role;

  getProtected(): boolean;
  setProtected(value: boolean): Role;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Role.AsObject;
  static toObject(includeInstance: boolean, msg: Role): Role.AsObject;
  static serializeBinaryToWriter(message: Role, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Role;
  static deserializeBinaryFromReader(message: Role, reader: jspb.BinaryReader): Role;
}

export namespace Role {
  export type AsObject = {
    id: string;
    displayName: string;
    description: string;
    permissionIdsList: Array<string>;
    legacyRoleName: string;
    pb_protected: boolean;
  };
}

export class RoleAssignment extends jspb.Message {
  getId(): string;
  setId(value: string): RoleAssignment;

  getAccountId(): string;
  setAccountId(value: string): RoleAssignment;

  getRoleId(): string;
  setRoleId(value: string): RoleAssignment;

  getScope(): RoleAssignment.Scope | undefined;
  setScope(value?: RoleAssignment.Scope): RoleAssignment;
  hasScope(): boolean;
  clearScope(): RoleAssignment;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RoleAssignment.AsObject;
  static toObject(includeInstance: boolean, msg: RoleAssignment): RoleAssignment.AsObject;
  static serializeBinaryToWriter(message: RoleAssignment, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RoleAssignment;
  static deserializeBinaryFromReader(message: RoleAssignment, reader: jspb.BinaryReader): RoleAssignment;
}

export namespace RoleAssignment {
  export type AsObject = {
    id: string;
    accountId: string;
    roleId: string;
    scope?: RoleAssignment.Scope.AsObject;
  };

  export class Scope extends jspb.Message {
    getResourceType(): RoleAssignment.ResourceType;
    setResourceType(value: RoleAssignment.ResourceType): Scope;

    getResource(): string;
    setResource(value: string): Scope;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Scope.AsObject;
    static toObject(includeInstance: boolean, msg: Scope): Scope.AsObject;
    static serializeBinaryToWriter(message: Scope, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Scope;
    static deserializeBinaryFromReader(message: Scope, reader: jspb.BinaryReader): Scope;
  }

  export namespace Scope {
    export type AsObject = {
      resourceType: RoleAssignment.ResourceType;
      resource: string;
    };
  }


  export enum ResourceType {
    RESOURCE_TYPE_UNSPECIFIED = 0,
    NAMED_RESOURCE = 1,
    NAMED_RESOURCE_PATH_PREFIX = 2,
    NODE = 3,
    SUBSYSTEM = 4,
    ZONE = 5,
  }
}

export class Permission extends jspb.Message {
  getId(): string;
  setId(value: string): Permission;

  getDisplayName(): string;
  setDisplayName(value: string): Permission;

  getDescription(): string;
  setDescription(value: string): Permission;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Permission.AsObject;
  static toObject(includeInstance: boolean, msg: Permission): Permission.AsObject;
  static serializeBinaryToWriter(message: Permission, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Permission;
  static deserializeBinaryFromReader(message: Permission, reader: jspb.BinaryReader): Permission;
}

export namespace Permission {
  export type AsObject = {
    id: string;
    displayName: string;
    description: string;
  };
}

export class AccountLimits extends jspb.Message {
  getUsername(): AccountLimits.Field | undefined;
  setUsername(value?: AccountLimits.Field): AccountLimits;
  hasUsername(): boolean;
  clearUsername(): AccountLimits;

  getPassword(): AccountLimits.Field | undefined;
  setPassword(value?: AccountLimits.Field): AccountLimits;
  hasPassword(): boolean;
  clearPassword(): AccountLimits;

  getDisplayName(): AccountLimits.Field | undefined;
  setDisplayName(value?: AccountLimits.Field): AccountLimits;
  hasDisplayName(): boolean;
  clearDisplayName(): AccountLimits;

  getDescription(): AccountLimits.Field | undefined;
  setDescription(value?: AccountLimits.Field): AccountLimits;
  hasDescription(): boolean;
  clearDescription(): AccountLimits;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AccountLimits.AsObject;
  static toObject(includeInstance: boolean, msg: AccountLimits): AccountLimits.AsObject;
  static serializeBinaryToWriter(message: AccountLimits, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AccountLimits;
  static deserializeBinaryFromReader(message: AccountLimits, reader: jspb.BinaryReader): AccountLimits;
}

export namespace AccountLimits {
  export type AsObject = {
    username?: AccountLimits.Field.AsObject;
    password?: AccountLimits.Field.AsObject;
    displayName?: AccountLimits.Field.AsObject;
    description?: AccountLimits.Field.AsObject;
  };

  export class Field extends jspb.Message {
    getMinLength(): number;
    setMinLength(value: number): Field;

    getMaxLength(): number;
    setMaxLength(value: number): Field;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Field.AsObject;
    static toObject(includeInstance: boolean, msg: Field): Field.AsObject;
    static serializeBinaryToWriter(message: Field, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Field;
    static deserializeBinaryFromReader(message: Field, reader: jspb.BinaryReader): Field;
  }

  export namespace Field {
    export type AsObject = {
      minLength: number;
      maxLength: number;
    };
  }

}

