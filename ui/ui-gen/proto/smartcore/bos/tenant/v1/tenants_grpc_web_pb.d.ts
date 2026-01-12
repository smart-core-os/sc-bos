import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_tenant_v1_tenants_pb from '../../../../smartcore/bos/tenant/v1/tenants_pb'; // proto import: "smartcore/bos/tenant/v1/tenants.proto"


export class TenantApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listTenants(
    request: smartcore_bos_tenant_v1_tenants_pb.ListTenantsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_tenant_v1_tenants_pb.ListTenantsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.ListTenantsResponse>;

  pullTenants(
    request: smartcore_bos_tenant_v1_tenants_pb.PullTenantsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.PullTenantsResponse>;

  createTenant(
    request: smartcore_bos_tenant_v1_tenants_pb.CreateTenantRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_tenant_v1_tenants_pb.Tenant) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.Tenant>;

  getTenant(
    request: smartcore_bos_tenant_v1_tenants_pb.GetTenantRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_tenant_v1_tenants_pb.Tenant) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.Tenant>;

  updateTenant(
    request: smartcore_bos_tenant_v1_tenants_pb.UpdateTenantRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_tenant_v1_tenants_pb.Tenant) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.Tenant>;

  deleteTenant(
    request: smartcore_bos_tenant_v1_tenants_pb.DeleteTenantRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_tenant_v1_tenants_pb.DeleteTenantResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.DeleteTenantResponse>;

  pullTenant(
    request: smartcore_bos_tenant_v1_tenants_pb.PullTenantRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.PullTenantResponse>;

  addTenantZones(
    request: smartcore_bos_tenant_v1_tenants_pb.AddTenantZonesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_tenant_v1_tenants_pb.Tenant) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.Tenant>;

  removeTenantZones(
    request: smartcore_bos_tenant_v1_tenants_pb.RemoveTenantZonesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_tenant_v1_tenants_pb.Tenant) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.Tenant>;

  listSecrets(
    request: smartcore_bos_tenant_v1_tenants_pb.ListSecretsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_tenant_v1_tenants_pb.ListSecretsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.ListSecretsResponse>;

  pullSecrets(
    request: smartcore_bos_tenant_v1_tenants_pb.PullSecretsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.PullSecretsResponse>;

  createSecret(
    request: smartcore_bos_tenant_v1_tenants_pb.CreateSecretRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_tenant_v1_tenants_pb.Secret) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.Secret>;

  verifySecret(
    request: smartcore_bos_tenant_v1_tenants_pb.VerifySecretRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_tenant_v1_tenants_pb.Secret) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.Secret>;

  getSecret(
    request: smartcore_bos_tenant_v1_tenants_pb.GetSecretRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_tenant_v1_tenants_pb.Secret) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.Secret>;

  updateSecret(
    request: smartcore_bos_tenant_v1_tenants_pb.UpdateSecretRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_tenant_v1_tenants_pb.Secret) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.Secret>;

  deleteSecret(
    request: smartcore_bos_tenant_v1_tenants_pb.DeleteSecretRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_tenant_v1_tenants_pb.DeleteSecretResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.DeleteSecretResponse>;

  pullSecret(
    request: smartcore_bos_tenant_v1_tenants_pb.PullSecretRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.PullSecretResponse>;

  regenerateSecret(
    request: smartcore_bos_tenant_v1_tenants_pb.RegenerateSecretRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_tenant_v1_tenants_pb.Secret) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.Secret>;

}

export class TenantApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listTenants(
    request: smartcore_bos_tenant_v1_tenants_pb.ListTenantsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_tenant_v1_tenants_pb.ListTenantsResponse>;

  pullTenants(
    request: smartcore_bos_tenant_v1_tenants_pb.PullTenantsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.PullTenantsResponse>;

  createTenant(
    request: smartcore_bos_tenant_v1_tenants_pb.CreateTenantRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_tenant_v1_tenants_pb.Tenant>;

  getTenant(
    request: smartcore_bos_tenant_v1_tenants_pb.GetTenantRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_tenant_v1_tenants_pb.Tenant>;

  updateTenant(
    request: smartcore_bos_tenant_v1_tenants_pb.UpdateTenantRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_tenant_v1_tenants_pb.Tenant>;

  deleteTenant(
    request: smartcore_bos_tenant_v1_tenants_pb.DeleteTenantRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_tenant_v1_tenants_pb.DeleteTenantResponse>;

  pullTenant(
    request: smartcore_bos_tenant_v1_tenants_pb.PullTenantRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.PullTenantResponse>;

  addTenantZones(
    request: smartcore_bos_tenant_v1_tenants_pb.AddTenantZonesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_tenant_v1_tenants_pb.Tenant>;

  removeTenantZones(
    request: smartcore_bos_tenant_v1_tenants_pb.RemoveTenantZonesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_tenant_v1_tenants_pb.Tenant>;

  listSecrets(
    request: smartcore_bos_tenant_v1_tenants_pb.ListSecretsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_tenant_v1_tenants_pb.ListSecretsResponse>;

  pullSecrets(
    request: smartcore_bos_tenant_v1_tenants_pb.PullSecretsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.PullSecretsResponse>;

  createSecret(
    request: smartcore_bos_tenant_v1_tenants_pb.CreateSecretRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_tenant_v1_tenants_pb.Secret>;

  verifySecret(
    request: smartcore_bos_tenant_v1_tenants_pb.VerifySecretRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_tenant_v1_tenants_pb.Secret>;

  getSecret(
    request: smartcore_bos_tenant_v1_tenants_pb.GetSecretRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_tenant_v1_tenants_pb.Secret>;

  updateSecret(
    request: smartcore_bos_tenant_v1_tenants_pb.UpdateSecretRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_tenant_v1_tenants_pb.Secret>;

  deleteSecret(
    request: smartcore_bos_tenant_v1_tenants_pb.DeleteSecretRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_tenant_v1_tenants_pb.DeleteSecretResponse>;

  pullSecret(
    request: smartcore_bos_tenant_v1_tenants_pb.PullSecretRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_tenant_v1_tenants_pb.PullSecretResponse>;

  regenerateSecret(
    request: smartcore_bos_tenant_v1_tenants_pb.RegenerateSecretRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_tenant_v1_tenants_pb.Secret>;

}

