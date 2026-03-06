import * as grpcWeb from 'grpc-web';

import * as traits_vending_pb from '../traits/vending_pb'; // proto import: "traits/vending.proto"


export class VendingApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listConsumables(
    request: traits_vending_pb.ListConsumablesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_vending_pb.ListConsumablesResponse) => void
  ): grpcWeb.ClientReadableStream<traits_vending_pb.ListConsumablesResponse>;

  pullConsumables(
    request: traits_vending_pb.PullConsumablesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_vending_pb.PullConsumablesResponse>;

  getStock(
    request: traits_vending_pb.GetStockRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_vending_pb.Consumable.Stock) => void
  ): grpcWeb.ClientReadableStream<traits_vending_pb.Consumable.Stock>;

  updateStock(
    request: traits_vending_pb.UpdateStockRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_vending_pb.Consumable.Stock) => void
  ): grpcWeb.ClientReadableStream<traits_vending_pb.Consumable.Stock>;

  pullStock(
    request: traits_vending_pb.PullStockRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_vending_pb.PullStockResponse>;

  listInventory(
    request: traits_vending_pb.ListInventoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_vending_pb.ListInventoryResponse) => void
  ): grpcWeb.ClientReadableStream<traits_vending_pb.ListInventoryResponse>;

  pullInventory(
    request: traits_vending_pb.PullInventoryRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_vending_pb.PullInventoryResponse>;

  dispense(
    request: traits_vending_pb.DispenseRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_vending_pb.Consumable.Stock) => void
  ): grpcWeb.ClientReadableStream<traits_vending_pb.Consumable.Stock>;

  stopDispense(
    request: traits_vending_pb.StopDispenseRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_vending_pb.Consumable.Stock) => void
  ): grpcWeb.ClientReadableStream<traits_vending_pb.Consumable.Stock>;

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
    request: traits_vending_pb.ListConsumablesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_vending_pb.ListConsumablesResponse>;

  pullConsumables(
    request: traits_vending_pb.PullConsumablesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_vending_pb.PullConsumablesResponse>;

  getStock(
    request: traits_vending_pb.GetStockRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_vending_pb.Consumable.Stock>;

  updateStock(
    request: traits_vending_pb.UpdateStockRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_vending_pb.Consumable.Stock>;

  pullStock(
    request: traits_vending_pb.PullStockRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_vending_pb.PullStockResponse>;

  listInventory(
    request: traits_vending_pb.ListInventoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_vending_pb.ListInventoryResponse>;

  pullInventory(
    request: traits_vending_pb.PullInventoryRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_vending_pb.PullInventoryResponse>;

  dispense(
    request: traits_vending_pb.DispenseRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_vending_pb.Consumable.Stock>;

  stopDispense(
    request: traits_vending_pb.StopDispenseRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_vending_pb.Consumable.Stock>;

}

export class VendingInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

