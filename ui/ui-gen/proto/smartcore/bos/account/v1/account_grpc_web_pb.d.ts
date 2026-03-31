import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_account_v1_account_pb from '../../../../smartcore/bos/account/v1/account_pb'; // proto import: "smartcore/bos/account/v1/account.proto"


export class AccountApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAccount(
    request: smartcore_bos_account_v1_account_pb.GetAccountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.Account) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.Account>;

  listAccounts(
    request: smartcore_bos_account_v1_account_pb.ListAccountsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.ListAccountsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.ListAccountsResponse>;

  createAccount(
    request: smartcore_bos_account_v1_account_pb.CreateAccountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.Account) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.Account>;

  updateAccount(
    request: smartcore_bos_account_v1_account_pb.UpdateAccountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.Account) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.Account>;

  updateAccountPassword(
    request: smartcore_bos_account_v1_account_pb.UpdateAccountPasswordRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.UpdateAccountPasswordResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.UpdateAccountPasswordResponse>;

  rotateAccountClientSecret(
    request: smartcore_bos_account_v1_account_pb.RotateAccountClientSecretRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.RotateAccountClientSecretResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.RotateAccountClientSecretResponse>;

  deleteAccount(
    request: smartcore_bos_account_v1_account_pb.DeleteAccountRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.DeleteAccountResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.DeleteAccountResponse>;

  getRole(
    request: smartcore_bos_account_v1_account_pb.GetRoleRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.Role) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.Role>;

  listRoles(
    request: smartcore_bos_account_v1_account_pb.ListRolesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.ListRolesResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.ListRolesResponse>;

  createRole(
    request: smartcore_bos_account_v1_account_pb.CreateRoleRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.Role) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.Role>;

  updateRole(
    request: smartcore_bos_account_v1_account_pb.UpdateRoleRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.Role) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.Role>;

  deleteRole(
    request: smartcore_bos_account_v1_account_pb.DeleteRoleRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.DeleteRoleResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.DeleteRoleResponse>;

  getRoleAssignment(
    request: smartcore_bos_account_v1_account_pb.GetRoleAssignmentRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.RoleAssignment) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.RoleAssignment>;

  listRoleAssignments(
    request: smartcore_bos_account_v1_account_pb.ListRoleAssignmentsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.ListRoleAssignmentsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.ListRoleAssignmentsResponse>;

  createRoleAssignment(
    request: smartcore_bos_account_v1_account_pb.CreateRoleAssignmentRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.RoleAssignment) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.RoleAssignment>;

  deleteRoleAssignment(
    request: smartcore_bos_account_v1_account_pb.DeleteRoleAssignmentRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.DeleteRoleAssignmentResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.DeleteRoleAssignmentResponse>;

}

export class AccountInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPermission(
    request: smartcore_bos_account_v1_account_pb.GetPermissionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.Permission) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.Permission>;

  listPermissions(
    request: smartcore_bos_account_v1_account_pb.ListPermissionsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.ListPermissionsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.ListPermissionsResponse>;

  getAccountLimits(
    request: smartcore_bos_account_v1_account_pb.GetAccountLimitsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_account_v1_account_pb.AccountLimits) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_account_v1_account_pb.AccountLimits>;

}

export class AccountApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getAccount(
    request: smartcore_bos_account_v1_account_pb.GetAccountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.Account>;

  listAccounts(
    request: smartcore_bos_account_v1_account_pb.ListAccountsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.ListAccountsResponse>;

  createAccount(
    request: smartcore_bos_account_v1_account_pb.CreateAccountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.Account>;

  updateAccount(
    request: smartcore_bos_account_v1_account_pb.UpdateAccountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.Account>;

  updateAccountPassword(
    request: smartcore_bos_account_v1_account_pb.UpdateAccountPasswordRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.UpdateAccountPasswordResponse>;

  rotateAccountClientSecret(
    request: smartcore_bos_account_v1_account_pb.RotateAccountClientSecretRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.RotateAccountClientSecretResponse>;

  deleteAccount(
    request: smartcore_bos_account_v1_account_pb.DeleteAccountRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.DeleteAccountResponse>;

  getRole(
    request: smartcore_bos_account_v1_account_pb.GetRoleRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.Role>;

  listRoles(
    request: smartcore_bos_account_v1_account_pb.ListRolesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.ListRolesResponse>;

  createRole(
    request: smartcore_bos_account_v1_account_pb.CreateRoleRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.Role>;

  updateRole(
    request: smartcore_bos_account_v1_account_pb.UpdateRoleRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.Role>;

  deleteRole(
    request: smartcore_bos_account_v1_account_pb.DeleteRoleRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.DeleteRoleResponse>;

  getRoleAssignment(
    request: smartcore_bos_account_v1_account_pb.GetRoleAssignmentRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.RoleAssignment>;

  listRoleAssignments(
    request: smartcore_bos_account_v1_account_pb.ListRoleAssignmentsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.ListRoleAssignmentsResponse>;

  createRoleAssignment(
    request: smartcore_bos_account_v1_account_pb.CreateRoleAssignmentRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.RoleAssignment>;

  deleteRoleAssignment(
    request: smartcore_bos_account_v1_account_pb.DeleteRoleAssignmentRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.DeleteRoleAssignmentResponse>;

}

export class AccountInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPermission(
    request: smartcore_bos_account_v1_account_pb.GetPermissionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.Permission>;

  listPermissions(
    request: smartcore_bos_account_v1_account_pb.ListPermissionsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.ListPermissionsResponse>;

  getAccountLimits(
    request: smartcore_bos_account_v1_account_pb.GetAccountLimitsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_account_v1_account_pb.AccountLimits>;

}

