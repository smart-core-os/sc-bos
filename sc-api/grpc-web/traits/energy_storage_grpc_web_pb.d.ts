import * as grpcWeb from 'grpc-web';

import * as traits_energy_storage_pb from '../traits/energy_storage_pb'; // proto import: "traits/energy_storage.proto"


export class EnergyStorageApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getEnergyLevel(
    request: traits_energy_storage_pb.GetEnergyLevelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_energy_storage_pb.EnergyLevel) => void
  ): grpcWeb.ClientReadableStream<traits_energy_storage_pb.EnergyLevel>;

  pullEnergyLevel(
    request: traits_energy_storage_pb.PullEnergyLevelRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_energy_storage_pb.PullEnergyLevelResponse>;

  charge(
    request: traits_energy_storage_pb.ChargeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_energy_storage_pb.ChargeResponse) => void
  ): grpcWeb.ClientReadableStream<traits_energy_storage_pb.ChargeResponse>;

}

export class EnergyStorageInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeEnergyLevel(
    request: traits_energy_storage_pb.DescribeEnergyLevelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_energy_storage_pb.EnergyLevelSupport) => void
  ): grpcWeb.ClientReadableStream<traits_energy_storage_pb.EnergyLevelSupport>;

}

export class EnergyStorageApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getEnergyLevel(
    request: traits_energy_storage_pb.GetEnergyLevelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_energy_storage_pb.EnergyLevel>;

  pullEnergyLevel(
    request: traits_energy_storage_pb.PullEnergyLevelRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_energy_storage_pb.PullEnergyLevelResponse>;

  charge(
    request: traits_energy_storage_pb.ChargeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_energy_storage_pb.ChargeResponse>;

}

export class EnergyStorageInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeEnergyLevel(
    request: traits_energy_storage_pb.DescribeEnergyLevelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_energy_storage_pb.EnergyLevelSupport>;

}

