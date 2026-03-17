import * as grpcWeb from 'grpc-web';

import * as traits_electric_pb from '../traits/electric_pb'; // proto import: "traits/electric.proto"


export class ElectricApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getDemand(
    request: traits_electric_pb.GetDemandRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_electric_pb.ElectricDemand) => void
  ): grpcWeb.ClientReadableStream<traits_electric_pb.ElectricDemand>;

  pullDemand(
    request: traits_electric_pb.PullDemandRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_electric_pb.PullDemandResponse>;

  getActiveMode(
    request: traits_electric_pb.GetActiveModeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_electric_pb.ElectricMode) => void
  ): grpcWeb.ClientReadableStream<traits_electric_pb.ElectricMode>;

  updateActiveMode(
    request: traits_electric_pb.UpdateActiveModeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_electric_pb.ElectricMode) => void
  ): grpcWeb.ClientReadableStream<traits_electric_pb.ElectricMode>;

  clearActiveMode(
    request: traits_electric_pb.ClearActiveModeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_electric_pb.ElectricMode) => void
  ): grpcWeb.ClientReadableStream<traits_electric_pb.ElectricMode>;

  pullActiveMode(
    request: traits_electric_pb.PullActiveModeRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_electric_pb.PullActiveModeResponse>;

  listModes(
    request: traits_electric_pb.ListModesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_electric_pb.ListModesResponse) => void
  ): grpcWeb.ClientReadableStream<traits_electric_pb.ListModesResponse>;

  pullModes(
    request: traits_electric_pb.PullModesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_electric_pb.PullModesResponse>;

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
    request: traits_electric_pb.GetDemandRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_electric_pb.ElectricDemand>;

  pullDemand(
    request: traits_electric_pb.PullDemandRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_electric_pb.PullDemandResponse>;

  getActiveMode(
    request: traits_electric_pb.GetActiveModeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_electric_pb.ElectricMode>;

  updateActiveMode(
    request: traits_electric_pb.UpdateActiveModeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_electric_pb.ElectricMode>;

  clearActiveMode(
    request: traits_electric_pb.ClearActiveModeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_electric_pb.ElectricMode>;

  pullActiveMode(
    request: traits_electric_pb.PullActiveModeRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_electric_pb.PullActiveModeResponse>;

  listModes(
    request: traits_electric_pb.ListModesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_electric_pb.ListModesResponse>;

  pullModes(
    request: traits_electric_pb.PullModesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_electric_pb.PullModesResponse>;

}

export class ElectricInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

