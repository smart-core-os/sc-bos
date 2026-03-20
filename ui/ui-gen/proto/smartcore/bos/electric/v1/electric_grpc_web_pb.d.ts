import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_electric_v1_electric_pb from '../../../../smartcore/bos/electric/v1/electric_pb'; // proto import: "smartcore/bos/electric/v1/electric.proto"


export class ElectricApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getDemand(
    request: smartcore_bos_electric_v1_electric_pb.GetDemandRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_electric_v1_electric_pb.ElectricDemand) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_pb.ElectricDemand>;

  pullDemand(
    request: smartcore_bos_electric_v1_electric_pb.PullDemandRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_pb.PullDemandResponse>;

  getActiveMode(
    request: smartcore_bos_electric_v1_electric_pb.GetActiveModeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_electric_v1_electric_pb.ElectricMode) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_pb.ElectricMode>;

  updateActiveMode(
    request: smartcore_bos_electric_v1_electric_pb.UpdateActiveModeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_electric_v1_electric_pb.ElectricMode) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_pb.ElectricMode>;

  clearActiveMode(
    request: smartcore_bos_electric_v1_electric_pb.ClearActiveModeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_electric_v1_electric_pb.ElectricMode) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_pb.ElectricMode>;

  pullActiveMode(
    request: smartcore_bos_electric_v1_electric_pb.PullActiveModeRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_pb.PullActiveModeResponse>;

  listModes(
    request: smartcore_bos_electric_v1_electric_pb.ListModesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_electric_v1_electric_pb.ListModesResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_pb.ListModesResponse>;

  pullModes(
    request: smartcore_bos_electric_v1_electric_pb.PullModesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_pb.PullModesResponse>;

}

export class ElectricInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

export class ElectricApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getDemand(
    request: smartcore_bos_electric_v1_electric_pb.GetDemandRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_electric_v1_electric_pb.ElectricDemand>;

  pullDemand(
    request: smartcore_bos_electric_v1_electric_pb.PullDemandRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_pb.PullDemandResponse>;

  getActiveMode(
    request: smartcore_bos_electric_v1_electric_pb.GetActiveModeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_electric_v1_electric_pb.ElectricMode>;

  updateActiveMode(
    request: smartcore_bos_electric_v1_electric_pb.UpdateActiveModeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_electric_v1_electric_pb.ElectricMode>;

  clearActiveMode(
    request: smartcore_bos_electric_v1_electric_pb.ClearActiveModeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_electric_v1_electric_pb.ElectricMode>;

  pullActiveMode(
    request: smartcore_bos_electric_v1_electric_pb.PullActiveModeRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_pb.PullActiveModeResponse>;

  listModes(
    request: smartcore_bos_electric_v1_electric_pb.ListModesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_electric_v1_electric_pb.ListModesResponse>;

  pullModes(
    request: smartcore_bos_electric_v1_electric_pb.PullModesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_pb.PullModesResponse>;

}

export class ElectricInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

