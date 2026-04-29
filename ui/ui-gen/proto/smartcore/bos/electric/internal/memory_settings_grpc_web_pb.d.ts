import * as grpcWeb from 'grpc-web';

import * as google_protobuf_empty_pb from 'google-protobuf/google/protobuf/empty_pb'; // proto import: "google/protobuf/empty.proto"
import * as smartcore_bos_electric_v1_electric_pb from '../../../../smartcore/bos/electric/v1/electric_pb'; // proto import: "smartcore/bos/electric/v1/electric.proto"
import * as smartcore_bos_electric_internal_memory_settings_pb from '../../../../smartcore/bos/electric/internal/memory_settings_pb'; // proto import: "smartcore/bos/electric/internal/memory_settings.proto"


export class MemorySettingsApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  updateDemand(
    request: smartcore_bos_electric_internal_memory_settings_pb.UpdateDemandRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_electric_v1_electric_pb.ElectricDemand) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_pb.ElectricDemand>;

  createMode(
    request: smartcore_bos_electric_internal_memory_settings_pb.CreateModeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_electric_v1_electric_pb.ElectricMode) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_pb.ElectricMode>;

  updateMode(
    request: smartcore_bos_electric_internal_memory_settings_pb.UpdateModeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_electric_v1_electric_pb.ElectricMode) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_electric_v1_electric_pb.ElectricMode>;

  deleteMode(
    request: smartcore_bos_electric_internal_memory_settings_pb.DeleteModeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: google_protobuf_empty_pb.Empty) => void
  ): grpcWeb.ClientReadableStream<google_protobuf_empty_pb.Empty>;

}

export class MemorySettingsApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  updateDemand(
    request: smartcore_bos_electric_internal_memory_settings_pb.UpdateDemandRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_electric_v1_electric_pb.ElectricDemand>;

  createMode(
    request: smartcore_bos_electric_internal_memory_settings_pb.CreateModeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_electric_v1_electric_pb.ElectricMode>;

  updateMode(
    request: smartcore_bos_electric_internal_memory_settings_pb.UpdateModeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_electric_v1_electric_pb.ElectricMode>;

  deleteMode(
    request: smartcore_bos_electric_internal_memory_settings_pb.DeleteModeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<google_protobuf_empty_pb.Empty>;

}

