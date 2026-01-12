import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_fluidflow_v1_fluid_flow_pb from '../../../../smartcore/bos/fluidflow/v1/fluid_flow_pb'; // proto import: "smartcore/bos/fluidflow/v1/fluid_flow.proto"


export class FluidFlowApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getFluidFlow(
    request: smartcore_bos_fluidflow_v1_fluid_flow_pb.GetFluidFlowRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_fluidflow_v1_fluid_flow_pb.FluidFlow) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_fluidflow_v1_fluid_flow_pb.FluidFlow>;

  pullFluidFlow(
    request: smartcore_bos_fluidflow_v1_fluid_flow_pb.PullFluidFlowRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_fluidflow_v1_fluid_flow_pb.PullFluidFlowResponse>;

  updateFluidFlow(
    request: smartcore_bos_fluidflow_v1_fluid_flow_pb.UpdateFluidFlowRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_fluidflow_v1_fluid_flow_pb.FluidFlow) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_fluidflow_v1_fluid_flow_pb.FluidFlow>;

}

export class FluidFlowInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeFluidFlow(
    request: smartcore_bos_fluidflow_v1_fluid_flow_pb.DescribeFluidFlowRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_fluidflow_v1_fluid_flow_pb.FluidFlowSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_fluidflow_v1_fluid_flow_pb.FluidFlowSupport>;

}

export class FluidFlowApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getFluidFlow(
    request: smartcore_bos_fluidflow_v1_fluid_flow_pb.GetFluidFlowRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_fluidflow_v1_fluid_flow_pb.FluidFlow>;

  pullFluidFlow(
    request: smartcore_bos_fluidflow_v1_fluid_flow_pb.PullFluidFlowRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_fluidflow_v1_fluid_flow_pb.PullFluidFlowResponse>;

  updateFluidFlow(
    request: smartcore_bos_fluidflow_v1_fluid_flow_pb.UpdateFluidFlowRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_fluidflow_v1_fluid_flow_pb.FluidFlow>;

}

export class FluidFlowInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeFluidFlow(
    request: smartcore_bos_fluidflow_v1_fluid_flow_pb.DescribeFluidFlowRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_fluidflow_v1_fluid_flow_pb.FluidFlowSupport>;

}

