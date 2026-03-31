import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_vending_v1_vending_pb from '../../../../smartcore/bos/vending/v1/vending_pb'; // proto import: "smartcore/bos/vending/v1/vending.proto"


export class VendingApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listConsumables(
    request: smartcore_bos_vending_v1_vending_pb.ListConsumablesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_vending_v1_vending_pb.ListConsumablesResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_vending_v1_vending_pb.ListConsumablesResponse>;

  pullConsumables(
    request: smartcore_bos_vending_v1_vending_pb.PullConsumablesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_vending_v1_vending_pb.PullConsumablesResponse>;

  getStock(
    request: smartcore_bos_vending_v1_vending_pb.GetStockRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_vending_v1_vending_pb.Consumable.Stock) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_vending_v1_vending_pb.Consumable.Stock>;

  updateStock(
    request: smartcore_bos_vending_v1_vending_pb.UpdateStockRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_vending_v1_vending_pb.Consumable.Stock) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_vending_v1_vending_pb.Consumable.Stock>;

  pullStock(
    request: smartcore_bos_vending_v1_vending_pb.PullStockRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_vending_v1_vending_pb.PullStockResponse>;

  listInventory(
    request: smartcore_bos_vending_v1_vending_pb.ListInventoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_vending_v1_vending_pb.ListInventoryResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_vending_v1_vending_pb.ListInventoryResponse>;

  pullInventory(
    request: smartcore_bos_vending_v1_vending_pb.PullInventoryRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_vending_v1_vending_pb.PullInventoryResponse>;

  dispense(
    request: smartcore_bos_vending_v1_vending_pb.DispenseRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_vending_v1_vending_pb.Consumable.Stock) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_vending_v1_vending_pb.Consumable.Stock>;

  stopDispense(
    request: smartcore_bos_vending_v1_vending_pb.StopDispenseRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_vending_v1_vending_pb.Consumable.Stock) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_vending_v1_vending_pb.Consumable.Stock>;

}

export class VendingInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

export class VendingApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listConsumables(
    request: smartcore_bos_vending_v1_vending_pb.ListConsumablesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_vending_v1_vending_pb.ListConsumablesResponse>;

  pullConsumables(
    request: smartcore_bos_vending_v1_vending_pb.PullConsumablesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_vending_v1_vending_pb.PullConsumablesResponse>;

  getStock(
    request: smartcore_bos_vending_v1_vending_pb.GetStockRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_vending_v1_vending_pb.Consumable.Stock>;

  updateStock(
    request: smartcore_bos_vending_v1_vending_pb.UpdateStockRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_vending_v1_vending_pb.Consumable.Stock>;

  pullStock(
    request: smartcore_bos_vending_v1_vending_pb.PullStockRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_vending_v1_vending_pb.PullStockResponse>;

  listInventory(
    request: smartcore_bos_vending_v1_vending_pb.ListInventoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_vending_v1_vending_pb.ListInventoryResponse>;

  pullInventory(
    request: smartcore_bos_vending_v1_vending_pb.PullInventoryRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_vending_v1_vending_pb.PullInventoryResponse>;

  dispense(
    request: smartcore_bos_vending_v1_vending_pb.DispenseRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_vending_v1_vending_pb.Consumable.Stock>;

  stopDispense(
    request: smartcore_bos_vending_v1_vending_pb.StopDispenseRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_vending_v1_vending_pb.Consumable.Stock>;

}

export class VendingInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

