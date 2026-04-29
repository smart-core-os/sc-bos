import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb from '../../../../../smartcore/bos/ops/cloud/v1alpha/cloud_connection_pb'; // proto import: "smartcore/bos/ops/cloud/v1alpha/cloud_connection.proto"


export class CloudConnectionApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getCloudConnection(
    request: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.GetCloudConnectionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.GetCloudConnectionResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.GetCloudConnectionResponse>;

  pullCloudConnection(
    request: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.PullCloudConnectionRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.PullCloudConnectionResponse>;

  registerCloudConnection(
    request: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.RegisterCloudConnectionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.RegisterCloudConnectionResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.RegisterCloudConnectionResponse>;

  unlinkCloudConnection(
    request: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.UnlinkCloudConnectionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.UnlinkCloudConnectionResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.UnlinkCloudConnectionResponse>;

  getCloudConnectionDefaults(
    request: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.GetCloudConnectionDefaultsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.GetCloudConnectionDefaultsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.GetCloudConnectionDefaultsResponse>;

  testCloudConnection(
    request: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.TestCloudConnectionRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.TestCloudConnectionResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.TestCloudConnectionResponse>;

}

export class CloudConnectionApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getCloudConnection(
    request: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.GetCloudConnectionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.GetCloudConnectionResponse>;

  pullCloudConnection(
    request: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.PullCloudConnectionRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.PullCloudConnectionResponse>;

  registerCloudConnection(
    request: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.RegisterCloudConnectionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.RegisterCloudConnectionResponse>;

  unlinkCloudConnection(
    request: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.UnlinkCloudConnectionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.UnlinkCloudConnectionResponse>;

  getCloudConnectionDefaults(
    request: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.GetCloudConnectionDefaultsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.GetCloudConnectionDefaultsResponse>;

  testCloudConnection(
    request: smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.TestCloudConnectionRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_ops_cloud_v1alpha_cloud_connection_pb.TestCloudConnectionResponse>;

}

