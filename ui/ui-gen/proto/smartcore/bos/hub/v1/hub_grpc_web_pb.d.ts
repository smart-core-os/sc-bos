import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_hub_v1_hub_pb from '../../../../smartcore/bos/hub/v1/hub_pb'; // proto import: "smartcore/bos/hub/v1/hub.proto"


export class HubApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getHubNode(
    request: smartcore_bos_hub_v1_hub_pb.GetHubNodeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_hub_v1_hub_pb.HubNode) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_hub_v1_hub_pb.HubNode>;

  listHubNodes(
    request: smartcore_bos_hub_v1_hub_pb.ListHubNodesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_hub_v1_hub_pb.ListHubNodesResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_hub_v1_hub_pb.ListHubNodesResponse>;

  pullHubNodes(
    request: smartcore_bos_hub_v1_hub_pb.PullHubNodesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_hub_v1_hub_pb.PullHubNodesResponse>;

  inspectHubNode(
    request: smartcore_bos_hub_v1_hub_pb.InspectHubNodeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_hub_v1_hub_pb.HubNodeInspection) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_hub_v1_hub_pb.HubNodeInspection>;

  enrollHubNode(
    request: smartcore_bos_hub_v1_hub_pb.EnrollHubNodeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_hub_v1_hub_pb.HubNode) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_hub_v1_hub_pb.HubNode>;

  renewHubNode(
    request: smartcore_bos_hub_v1_hub_pb.RenewHubNodeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_hub_v1_hub_pb.HubNode) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_hub_v1_hub_pb.HubNode>;

  testHubNode(
    request: smartcore_bos_hub_v1_hub_pb.TestHubNodeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_hub_v1_hub_pb.TestHubNodeResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_hub_v1_hub_pb.TestHubNodeResponse>;

  forgetHubNode(
    request: smartcore_bos_hub_v1_hub_pb.ForgetHubNodeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_hub_v1_hub_pb.ForgetHubNodeResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_hub_v1_hub_pb.ForgetHubNodeResponse>;

}

export class HubApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getHubNode(
    request: smartcore_bos_hub_v1_hub_pb.GetHubNodeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_hub_v1_hub_pb.HubNode>;

  listHubNodes(
    request: smartcore_bos_hub_v1_hub_pb.ListHubNodesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_hub_v1_hub_pb.ListHubNodesResponse>;

  pullHubNodes(
    request: smartcore_bos_hub_v1_hub_pb.PullHubNodesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_hub_v1_hub_pb.PullHubNodesResponse>;

  inspectHubNode(
    request: smartcore_bos_hub_v1_hub_pb.InspectHubNodeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_hub_v1_hub_pb.HubNodeInspection>;

  enrollHubNode(
    request: smartcore_bos_hub_v1_hub_pb.EnrollHubNodeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_hub_v1_hub_pb.HubNode>;

  renewHubNode(
    request: smartcore_bos_hub_v1_hub_pb.RenewHubNodeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_hub_v1_hub_pb.HubNode>;

  testHubNode(
    request: smartcore_bos_hub_v1_hub_pb.TestHubNodeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_hub_v1_hub_pb.TestHubNodeResponse>;

  forgetHubNode(
    request: smartcore_bos_hub_v1_hub_pb.ForgetHubNodeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_hub_v1_hub_pb.ForgetHubNodeResponse>;

}

