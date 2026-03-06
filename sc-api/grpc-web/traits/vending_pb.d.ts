import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_change_pb from '../types/change_pb'; // proto import: "types/change.proto"
import * as types_image_pb from '../types/image_pb'; // proto import: "types/image.proto"
import * as types_number_pb from '../types/number_pb'; // proto import: "types/number.proto"


export class Consumable extends jspb.Message {
  getName(): string;
  setName(value: string): Consumable;

  getAvailablePortionsList(): Array<Consumable.Portion>;
  setAvailablePortionsList(value: Array<Consumable.Portion>): Consumable;
  clearAvailablePortionsList(): Consumable;
  addAvailablePortions(value?: Consumable.Portion, index?: number): Consumable.Portion;

  getDefaultPortion(): Consumable.Quantity | undefined;
  setDefaultPortion(value?: Consumable.Quantity): Consumable;
  hasDefaultPortion(): boolean;
  clearDefaultPortion(): Consumable;

  getTitle(): string;
  setTitle(value: string): Consumable;

  getDisplayName(): string;
  setDisplayName(value: string): Consumable;

  getPicture(): types_image_pb.Image | undefined;
  setPicture(value?: types_image_pb.Image): Consumable;
  hasPicture(): boolean;
  clearPicture(): Consumable;

  getUrl(): string;
  setUrl(value: string): Consumable;

  getIdsMap(): jspb.Map<string, string>;
  clearIdsMap(): Consumable;

  getMoreMap(): jspb.Map<string, string>;
  clearMoreMap(): Consumable;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Consumable.AsObject;
  static toObject(includeInstance: boolean, msg: Consumable): Consumable.AsObject;
  static serializeBinaryToWriter(message: Consumable, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Consumable;
  static deserializeBinaryFromReader(message: Consumable, reader: jspb.BinaryReader): Consumable;
}

export namespace Consumable {
  export type AsObject = {
    name: string;
    availablePortionsList: Array<Consumable.Portion.AsObject>;
    defaultPortion?: Consumable.Quantity.AsObject;
    title: string;
    displayName: string;
    picture?: types_image_pb.Image.AsObject;
    url: string;
    idsMap: Array<[string, string]>;
    moreMap: Array<[string, string]>;
  };

  export class Portion extends jspb.Message {
    getUnit(): Consumable.Unit;
    setUnit(value: Consumable.Unit): Portion;

    getBounds(): types_number_pb.FloatBounds | undefined;
    setBounds(value?: types_number_pb.FloatBounds): Portion;
    hasBounds(): boolean;
    clearBounds(): Portion;

    getStep(): number;
    setStep(value: number): Portion;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Portion.AsObject;
    static toObject(includeInstance: boolean, msg: Portion): Portion.AsObject;
    static serializeBinaryToWriter(message: Portion, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Portion;
    static deserializeBinaryFromReader(message: Portion, reader: jspb.BinaryReader): Portion;
  }

  export namespace Portion {
    export type AsObject = {
      unit: Consumable.Unit;
      bounds?: types_number_pb.FloatBounds.AsObject;
      step: number;
    };
  }


  export class Quantity extends jspb.Message {
    getAmount(): number;
    setAmount(value: number): Quantity;

    getUnit(): Consumable.Unit;
    setUnit(value: Consumable.Unit): Quantity;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Quantity.AsObject;
    static toObject(includeInstance: boolean, msg: Quantity): Quantity.AsObject;
    static serializeBinaryToWriter(message: Quantity, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Quantity;
    static deserializeBinaryFromReader(message: Quantity, reader: jspb.BinaryReader): Quantity;
  }

  export namespace Quantity {
    export type AsObject = {
      amount: number;
      unit: Consumable.Unit;
    };
  }


  export class Stock extends jspb.Message {
    getConsumable(): string;
    setConsumable(value: string): Stock;

    getRemaining(): Consumable.Quantity | undefined;
    setRemaining(value?: Consumable.Quantity): Stock;
    hasRemaining(): boolean;
    clearRemaining(): Stock;

    getUsed(): Consumable.Quantity | undefined;
    setUsed(value?: Consumable.Quantity): Stock;
    hasUsed(): boolean;
    clearUsed(): Stock;

    getLastDispensed(): Consumable.Quantity | undefined;
    setLastDispensed(value?: Consumable.Quantity): Stock;
    hasLastDispensed(): boolean;
    clearLastDispensed(): Stock;

    getDispensing(): boolean;
    setDispensing(value: boolean): Stock;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Stock.AsObject;
    static toObject(includeInstance: boolean, msg: Stock): Stock.AsObject;
    static serializeBinaryToWriter(message: Stock, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Stock;
    static deserializeBinaryFromReader(message: Stock, reader: jspb.BinaryReader): Stock;
  }

  export namespace Stock {
    export type AsObject = {
      consumable: string;
      remaining?: Consumable.Quantity.AsObject;
      used?: Consumable.Quantity.AsObject;
      lastDispensed?: Consumable.Quantity.AsObject;
      dispensing: boolean;
    };
  }


  export enum Unit {
    UNIT_UNSPECIFIED = 0,
    NO_UNIT = 1,
    METER = 2,
    LITER = 3,
    CUBIC_METER = 4,
    CUP = 5,
    KILOGRAM = 6,
  }
}

export class ListConsumablesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListConsumablesRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListConsumablesRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListConsumablesRequest;

  getPageSize(): number;
  setPageSize(value: number): ListConsumablesRequest;

  getPageToken(): string;
  setPageToken(value: string): ListConsumablesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListConsumablesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListConsumablesRequest): ListConsumablesRequest.AsObject;
  static serializeBinaryToWriter(message: ListConsumablesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListConsumablesRequest;
  static deserializeBinaryFromReader(message: ListConsumablesRequest, reader: jspb.BinaryReader): ListConsumablesRequest;
}

export namespace ListConsumablesRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
  };
}

export class ListConsumablesResponse extends jspb.Message {
  getConsumablesList(): Array<Consumable>;
  setConsumablesList(value: Array<Consumable>): ListConsumablesResponse;
  clearConsumablesList(): ListConsumablesResponse;
  addConsumables(value?: Consumable, index?: number): Consumable;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListConsumablesResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListConsumablesResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListConsumablesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListConsumablesResponse): ListConsumablesResponse.AsObject;
  static serializeBinaryToWriter(message: ListConsumablesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListConsumablesResponse;
  static deserializeBinaryFromReader(message: ListConsumablesResponse, reader: jspb.BinaryReader): ListConsumablesResponse;
}

export namespace ListConsumablesResponse {
  export type AsObject = {
    consumablesList: Array<Consumable.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class PullConsumablesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullConsumablesRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullConsumablesRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullConsumablesRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullConsumablesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullConsumablesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullConsumablesRequest): PullConsumablesRequest.AsObject;
  static serializeBinaryToWriter(message: PullConsumablesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullConsumablesRequest;
  static deserializeBinaryFromReader(message: PullConsumablesRequest, reader: jspb.BinaryReader): PullConsumablesRequest;
}

export namespace PullConsumablesRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullConsumablesResponse extends jspb.Message {
  getChangesList(): Array<PullConsumablesResponse.Change>;
  setChangesList(value: Array<PullConsumablesResponse.Change>): PullConsumablesResponse;
  clearChangesList(): PullConsumablesResponse;
  addChanges(value?: PullConsumablesResponse.Change, index?: number): PullConsumablesResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullConsumablesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullConsumablesResponse): PullConsumablesResponse.AsObject;
  static serializeBinaryToWriter(message: PullConsumablesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullConsumablesResponse;
  static deserializeBinaryFromReader(message: PullConsumablesResponse, reader: jspb.BinaryReader): PullConsumablesResponse;
}

export namespace PullConsumablesResponse {
  export type AsObject = {
    changesList: Array<PullConsumablesResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getType(): types_change_pb.ChangeType;
    setType(value: types_change_pb.ChangeType): Change;

    getNewValue(): Consumable | undefined;
    setNewValue(value?: Consumable): Change;
    hasNewValue(): boolean;
    clearNewValue(): Change;

    getOldValue(): Consumable | undefined;
    setOldValue(value?: Consumable): Change;
    hasOldValue(): boolean;
    clearOldValue(): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Change.AsObject;
    static toObject(includeInstance: boolean, msg: Change): Change.AsObject;
    static serializeBinaryToWriter(message: Change, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Change;
    static deserializeBinaryFromReader(message: Change, reader: jspb.BinaryReader): Change;
  }

  export namespace Change {
    export type AsObject = {
      name: string;
      type: types_change_pb.ChangeType;
      newValue?: Consumable.AsObject;
      oldValue?: Consumable.AsObject;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };
  }

}

export class GetStockRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetStockRequest;

  getConsumable(): string;
  setConsumable(value: string): GetStockRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetStockRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetStockRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetStockRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetStockRequest): GetStockRequest.AsObject;
  static serializeBinaryToWriter(message: GetStockRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetStockRequest;
  static deserializeBinaryFromReader(message: GetStockRequest, reader: jspb.BinaryReader): GetStockRequest;
}

export namespace GetStockRequest {
  export type AsObject = {
    name: string;
    consumable: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateStockRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateStockRequest;

  getStock(): Consumable.Stock | undefined;
  setStock(value?: Consumable.Stock): UpdateStockRequest;
  hasStock(): boolean;
  clearStock(): UpdateStockRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateStockRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateStockRequest;

  getRelative(): boolean;
  setRelative(value: boolean): UpdateStockRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateStockRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateStockRequest): UpdateStockRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateStockRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateStockRequest;
  static deserializeBinaryFromReader(message: UpdateStockRequest, reader: jspb.BinaryReader): UpdateStockRequest;
}

export namespace UpdateStockRequest {
  export type AsObject = {
    name: string;
    stock?: Consumable.Stock.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    relative: boolean;
  };
}

export class PullStockRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullStockRequest;

  getConsumable(): string;
  setConsumable(value: string): PullStockRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullStockRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullStockRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullStockRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullStockRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullStockRequest): PullStockRequest.AsObject;
  static serializeBinaryToWriter(message: PullStockRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullStockRequest;
  static deserializeBinaryFromReader(message: PullStockRequest, reader: jspb.BinaryReader): PullStockRequest;
}

export namespace PullStockRequest {
  export type AsObject = {
    name: string;
    consumable: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullStockResponse extends jspb.Message {
  getChangesList(): Array<PullStockResponse.Change>;
  setChangesList(value: Array<PullStockResponse.Change>): PullStockResponse;
  clearChangesList(): PullStockResponse;
  addChanges(value?: PullStockResponse.Change, index?: number): PullStockResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullStockResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullStockResponse): PullStockResponse.AsObject;
  static serializeBinaryToWriter(message: PullStockResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullStockResponse;
  static deserializeBinaryFromReader(message: PullStockResponse, reader: jspb.BinaryReader): PullStockResponse;
}

export namespace PullStockResponse {
  export type AsObject = {
    changesList: Array<PullStockResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getStock(): Consumable.Stock | undefined;
    setStock(value?: Consumable.Stock): Change;
    hasStock(): boolean;
    clearStock(): Change;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Change.AsObject;
    static toObject(includeInstance: boolean, msg: Change): Change.AsObject;
    static serializeBinaryToWriter(message: Change, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Change;
    static deserializeBinaryFromReader(message: Change, reader: jspb.BinaryReader): Change;
  }

  export namespace Change {
    export type AsObject = {
      name: string;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      stock?: Consumable.Stock.AsObject;
    };
  }

}

export class ListInventoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListInventoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListInventoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListInventoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListInventoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListInventoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListInventoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListInventoryRequest): ListInventoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListInventoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListInventoryRequest;
  static deserializeBinaryFromReader(message: ListInventoryRequest, reader: jspb.BinaryReader): ListInventoryRequest;
}

export namespace ListInventoryRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
  };
}

export class ListInventoryResponse extends jspb.Message {
  getInventoryList(): Array<Consumable.Stock>;
  setInventoryList(value: Array<Consumable.Stock>): ListInventoryResponse;
  clearInventoryList(): ListInventoryResponse;
  addInventory(value?: Consumable.Stock, index?: number): Consumable.Stock;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListInventoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListInventoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListInventoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListInventoryResponse): ListInventoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListInventoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListInventoryResponse;
  static deserializeBinaryFromReader(message: ListInventoryResponse, reader: jspb.BinaryReader): ListInventoryResponse;
}

export namespace ListInventoryResponse {
  export type AsObject = {
    inventoryList: Array<Consumable.Stock.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class PullInventoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullInventoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullInventoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullInventoryRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullInventoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullInventoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullInventoryRequest): PullInventoryRequest.AsObject;
  static serializeBinaryToWriter(message: PullInventoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullInventoryRequest;
  static deserializeBinaryFromReader(message: PullInventoryRequest, reader: jspb.BinaryReader): PullInventoryRequest;
}

export namespace PullInventoryRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullInventoryResponse extends jspb.Message {
  getChangesList(): Array<PullInventoryResponse.Change>;
  setChangesList(value: Array<PullInventoryResponse.Change>): PullInventoryResponse;
  clearChangesList(): PullInventoryResponse;
  addChanges(value?: PullInventoryResponse.Change, index?: number): PullInventoryResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullInventoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullInventoryResponse): PullInventoryResponse.AsObject;
  static serializeBinaryToWriter(message: PullInventoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullInventoryResponse;
  static deserializeBinaryFromReader(message: PullInventoryResponse, reader: jspb.BinaryReader): PullInventoryResponse;
}

export namespace PullInventoryResponse {
  export type AsObject = {
    changesList: Array<PullInventoryResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getType(): types_change_pb.ChangeType;
    setType(value: types_change_pb.ChangeType): Change;

    getNewValue(): Consumable.Stock | undefined;
    setNewValue(value?: Consumable.Stock): Change;
    hasNewValue(): boolean;
    clearNewValue(): Change;

    getOldValue(): Consumable.Stock | undefined;
    setOldValue(value?: Consumable.Stock): Change;
    hasOldValue(): boolean;
    clearOldValue(): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Change.AsObject;
    static toObject(includeInstance: boolean, msg: Change): Change.AsObject;
    static serializeBinaryToWriter(message: Change, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Change;
    static deserializeBinaryFromReader(message: Change, reader: jspb.BinaryReader): Change;
  }

  export namespace Change {
    export type AsObject = {
      name: string;
      type: types_change_pb.ChangeType;
      newValue?: Consumable.Stock.AsObject;
      oldValue?: Consumable.Stock.AsObject;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };
  }

}

export class DispenseRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DispenseRequest;

  getConsumable(): string;
  setConsumable(value: string): DispenseRequest;

  getQuantity(): Consumable.Quantity | undefined;
  setQuantity(value?: Consumable.Quantity): DispenseRequest;
  hasQuantity(): boolean;
  clearQuantity(): DispenseRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): DispenseRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): DispenseRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DispenseRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DispenseRequest): DispenseRequest.AsObject;
  static serializeBinaryToWriter(message: DispenseRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DispenseRequest;
  static deserializeBinaryFromReader(message: DispenseRequest, reader: jspb.BinaryReader): DispenseRequest;
}

export namespace DispenseRequest {
  export type AsObject = {
    name: string;
    consumable: string;
    quantity?: Consumable.Quantity.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class StopDispenseRequest extends jspb.Message {
  getName(): string;
  setName(value: string): StopDispenseRequest;

  getConsumable(): string;
  setConsumable(value: string): StopDispenseRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StopDispenseRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StopDispenseRequest): StopDispenseRequest.AsObject;
  static serializeBinaryToWriter(message: StopDispenseRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StopDispenseRequest;
  static deserializeBinaryFromReader(message: StopDispenseRequest, reader: jspb.BinaryReader): StopDispenseRequest;
}

export namespace StopDispenseRequest {
  export type AsObject = {
    name: string;
    consumable: string;
  };
}

