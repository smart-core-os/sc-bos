import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_accesscredential_v1_access_credential_pb from '../../../../smartcore/bos/accesscredential/v1/access_credential_pb'; // proto import: "smartcore/bos/accesscredential/v1/access_credential.proto"


export class AccessCredentialApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getCredential(
    request: smartcore_bos_accesscredential_v1_access_credential_pb.GetCredentialRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_accesscredential_v1_access_credential_pb.Credential) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_accesscredential_v1_access_credential_pb.Credential>;

  listCredentials(
    request: smartcore_bos_accesscredential_v1_access_credential_pb.ListCredentialsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_accesscredential_v1_access_credential_pb.ListCredentialsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_accesscredential_v1_access_credential_pb.ListCredentialsResponse>;

  createCredential(
    request: smartcore_bos_accesscredential_v1_access_credential_pb.CreateCredentialRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_accesscredential_v1_access_credential_pb.Credential) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_accesscredential_v1_access_credential_pb.Credential>;

  updateCredential(
    request: smartcore_bos_accesscredential_v1_access_credential_pb.UpdateCredentialRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_accesscredential_v1_access_credential_pb.Credential) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_accesscredential_v1_access_credential_pb.Credential>;

  deleteCredential(
    request: smartcore_bos_accesscredential_v1_access_credential_pb.DeleteCredentialRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_accesscredential_v1_access_credential_pb.DeleteCredentialResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_accesscredential_v1_access_credential_pb.DeleteCredentialResponse>;

}

export class AccessCredentialInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeCredential(
    request: smartcore_bos_accesscredential_v1_access_credential_pb.DescribeCredentialRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_accesscredential_v1_access_credential_pb.CredentialSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_accesscredential_v1_access_credential_pb.CredentialSupport>;

}

export class AccessCredentialApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getCredential(
    request: smartcore_bos_accesscredential_v1_access_credential_pb.GetCredentialRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_accesscredential_v1_access_credential_pb.Credential>;

  listCredentials(
    request: smartcore_bos_accesscredential_v1_access_credential_pb.ListCredentialsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_accesscredential_v1_access_credential_pb.ListCredentialsResponse>;

  createCredential(
    request: smartcore_bos_accesscredential_v1_access_credential_pb.CreateCredentialRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_accesscredential_v1_access_credential_pb.Credential>;

  updateCredential(
    request: smartcore_bos_accesscredential_v1_access_credential_pb.UpdateCredentialRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_accesscredential_v1_access_credential_pb.Credential>;

  deleteCredential(
    request: smartcore_bos_accesscredential_v1_access_credential_pb.DeleteCredentialRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_accesscredential_v1_access_credential_pb.DeleteCredentialResponse>;

}

export class AccessCredentialInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeCredential(
    request: smartcore_bos_accesscredential_v1_access_credential_pb.DescribeCredentialRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_accesscredential_v1_access_credential_pb.CredentialSupport>;

}

