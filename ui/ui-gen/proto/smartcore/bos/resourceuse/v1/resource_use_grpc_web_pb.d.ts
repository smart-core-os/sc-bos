import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_resourceuse_v1_resource_use_pb from '../../../../smartcore/bos/resourceuse/v1/resource_use_pb'; // proto import: "smartcore/bos/resourceuse/v1/resource_use.proto"


export class ResourceUseApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getResourceUse(
    request: smartcore_bos_resourceuse_v1_resource_use_pb.GetResourceUseRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_resourceuse_v1_resource_use_pb.ResourceUse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_resourceuse_v1_resource_use_pb.ResourceUse>;

  pullResourceUse(
    request: smartcore_bos_resourceuse_v1_resource_use_pb.PullResourceUseRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_resourceuse_v1_resource_use_pb.PullResourceUseResponse>;

}

export class ResourceUseApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getResourceUse(
    request: smartcore_bos_resourceuse_v1_resource_use_pb.GetResourceUseRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_resourceuse_v1_resource_use_pb.ResourceUse>;

  pullResourceUse(
    request: smartcore_bos_resourceuse_v1_resource_use_pb.PullResourceUseRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_resourceuse_v1_resource_use_pb.PullResourceUseResponse>;

}

