import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_resourceutilisation_v1_resource_utilisation_pb from '../../../../smartcore/bos/resourceutilisation/v1/resource_utilisation_pb'; // proto import: "smartcore/bos/resourceutilisation/v1/resource_utilisation.proto"


export class ResourceUtilisationApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getResourceUtilisation(
    request: smartcore_bos_resourceutilisation_v1_resource_utilisation_pb.GetResourceUtilisationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_resourceutilisation_v1_resource_utilisation_pb.ResourceUtilisation) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_resourceutilisation_v1_resource_utilisation_pb.ResourceUtilisation>;

  pullResourceUtilisation(
    request: smartcore_bos_resourceutilisation_v1_resource_utilisation_pb.PullResourceUtilisationRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_resourceutilisation_v1_resource_utilisation_pb.PullResourceUtilisationResponse>;

}

export class ResourceUtilisationApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getResourceUtilisation(
    request: smartcore_bos_resourceutilisation_v1_resource_utilisation_pb.GetResourceUtilisationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_resourceutilisation_v1_resource_utilisation_pb.ResourceUtilisation>;

  pullResourceUtilisation(
    request: smartcore_bos_resourceutilisation_v1_resource_utilisation_pb.PullResourceUtilisationRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_resourceutilisation_v1_resource_utilisation_pb.PullResourceUtilisationResponse>;

}

