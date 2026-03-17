import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_storage_v1_storage_pb from '../../../../smartcore/bos/storage/v1/storage_pb'; // proto import: "smartcore/bos/storage/v1/storage.proto"


export class StorageApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getStorage(
    request: smartcore_bos_storage_v1_storage_pb.GetStorageRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_storage_v1_storage_pb.Storage) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_storage_v1_storage_pb.Storage>;

  pullStorage(
    request: smartcore_bos_storage_v1_storage_pb.PullStorageRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_storage_v1_storage_pb.PullStorageResponse>;

}

export class StorageAdminApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  performStorageAdmin(
    request: smartcore_bos_storage_v1_storage_pb.PerformStorageAdminRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_storage_v1_storage_pb.PerformStorageAdminResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_storage_v1_storage_pb.PerformStorageAdminResponse>;

}

export class StorageInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeStorage(
    request: smartcore_bos_storage_v1_storage_pb.DescribeStorageRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_storage_v1_storage_pb.StorageSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_storage_v1_storage_pb.StorageSupport>;

}

export class StorageApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getStorage(
    request: smartcore_bos_storage_v1_storage_pb.GetStorageRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_storage_v1_storage_pb.Storage>;

  pullStorage(
    request: smartcore_bos_storage_v1_storage_pb.PullStorageRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_storage_v1_storage_pb.PullStorageResponse>;

}

export class StorageAdminApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  performStorageAdmin(
    request: smartcore_bos_storage_v1_storage_pb.PerformStorageAdminRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_storage_v1_storage_pb.PerformStorageAdminResponse>;

}

export class StorageInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeStorage(
    request: smartcore_bos_storage_v1_storage_pb.DescribeStorageRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_storage_v1_storage_pb.StorageSupport>;

}

