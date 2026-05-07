import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_data_retention_v1_data_retention_pb from '../../../../smartcore/bos/data_retention/v1/data_retention_pb'; // proto import: "smartcore/bos/data_retention/v1/data_retention.proto"


export class DataRetentionApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getDataRetention(
    request: smartcore_bos_data_retention_v1_data_retention_pb.GetDataRetentionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_data_retention_v1_data_retention_pb.DataRetention) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_data_retention_v1_data_retention_pb.DataRetention>;

  pullDataRetention(
    request: smartcore_bos_data_retention_v1_data_retention_pb.PullDataRetentionRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_data_retention_v1_data_retention_pb.PullDataRetentionResponse>;

  clearDataRetention(
    request: smartcore_bos_data_retention_v1_data_retention_pb.ClearDataRetentionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_data_retention_v1_data_retention_pb.ClearDataRetentionResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_data_retention_v1_data_retention_pb.ClearDataRetentionResponse>;

  deleteOldDataRetention(
    request: smartcore_bos_data_retention_v1_data_retention_pb.DeleteOldDataRetentionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_data_retention_v1_data_retention_pb.DeleteOldDataRetentionResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_data_retention_v1_data_retention_pb.DeleteOldDataRetentionResponse>;

  compactDataRetention(
    request: smartcore_bos_data_retention_v1_data_retention_pb.CompactDataRetentionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_data_retention_v1_data_retention_pb.CompactDataRetentionResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_data_retention_v1_data_retention_pb.CompactDataRetentionResponse>;

  springCleanDataRetention(
    request: smartcore_bos_data_retention_v1_data_retention_pb.SpringCleanDataRetentionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_data_retention_v1_data_retention_pb.SpringCleanDataRetentionResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_data_retention_v1_data_retention_pb.SpringCleanDataRetentionResponse>;

}

export class DataRetentionInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeDataRetention(
    request: smartcore_bos_data_retention_v1_data_retention_pb.DescribeDataRetentionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_data_retention_v1_data_retention_pb.DataRetentionSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_data_retention_v1_data_retention_pb.DataRetentionSupport>;

}

export class DataRetentionApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getDataRetention(
    request: smartcore_bos_data_retention_v1_data_retention_pb.GetDataRetentionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_data_retention_v1_data_retention_pb.DataRetention>;

  pullDataRetention(
    request: smartcore_bos_data_retention_v1_data_retention_pb.PullDataRetentionRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_data_retention_v1_data_retention_pb.PullDataRetentionResponse>;

  clearDataRetention(
    request: smartcore_bos_data_retention_v1_data_retention_pb.ClearDataRetentionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_data_retention_v1_data_retention_pb.ClearDataRetentionResponse>;

  deleteOldDataRetention(
    request: smartcore_bos_data_retention_v1_data_retention_pb.DeleteOldDataRetentionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_data_retention_v1_data_retention_pb.DeleteOldDataRetentionResponse>;

  compactDataRetention(
    request: smartcore_bos_data_retention_v1_data_retention_pb.CompactDataRetentionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_data_retention_v1_data_retention_pb.CompactDataRetentionResponse>;

  springCleanDataRetention(
    request: smartcore_bos_data_retention_v1_data_retention_pb.SpringCleanDataRetentionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_data_retention_v1_data_retention_pb.SpringCleanDataRetentionResponse>;

}

export class DataRetentionInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeDataRetention(
    request: smartcore_bos_data_retention_v1_data_retention_pb.DescribeDataRetentionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_data_retention_v1_data_retention_pb.DataRetentionSupport>;

}

