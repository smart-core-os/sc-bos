import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_energystorage_v1_energy_storage_pb from '../../../../smartcore/bos/energystorage/v1/energy_storage_pb'; // proto import: "smartcore/bos/energystorage/v1/energy_storage.proto"


export class EnergyStorageApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getEnergyLevel(
    request: smartcore_bos_energystorage_v1_energy_storage_pb.GetEnergyLevelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_energystorage_v1_energy_storage_pb.EnergyLevel) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_energystorage_v1_energy_storage_pb.EnergyLevel>;

  pullEnergyLevel(
    request: smartcore_bos_energystorage_v1_energy_storage_pb.PullEnergyLevelRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_energystorage_v1_energy_storage_pb.PullEnergyLevelResponse>;

  charge(
    request: smartcore_bos_energystorage_v1_energy_storage_pb.ChargeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_energystorage_v1_energy_storage_pb.ChargeResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_energystorage_v1_energy_storage_pb.ChargeResponse>;

}

export class EnergyStorageInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeEnergyLevel(
    request: smartcore_bos_energystorage_v1_energy_storage_pb.DescribeEnergyLevelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_energystorage_v1_energy_storage_pb.EnergyLevelSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_energystorage_v1_energy_storage_pb.EnergyLevelSupport>;

}

export class EnergyStorageApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getEnergyLevel(
    request: smartcore_bos_energystorage_v1_energy_storage_pb.GetEnergyLevelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_energystorage_v1_energy_storage_pb.EnergyLevel>;

  pullEnergyLevel(
    request: smartcore_bos_energystorage_v1_energy_storage_pb.PullEnergyLevelRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_energystorage_v1_energy_storage_pb.PullEnergyLevelResponse>;

  charge(
    request: smartcore_bos_energystorage_v1_energy_storage_pb.ChargeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_energystorage_v1_energy_storage_pb.ChargeResponse>;

}

export class EnergyStorageInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeEnergyLevel(
    request: smartcore_bos_energystorage_v1_energy_storage_pb.DescribeEnergyLevelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_energystorage_v1_energy_storage_pb.EnergyLevelSupport>;

}

