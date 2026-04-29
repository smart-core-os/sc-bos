import * as jspb from 'google-protobuf'

import * as google_protobuf_empty_pb from 'google-protobuf/google/protobuf/empty_pb'; // proto import: "google/protobuf/empty.proto"
import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as smartcore_bos_electric_v1_electric_pb from '../../../../smartcore/bos/electric/v1/electric_pb'; // proto import: "smartcore/bos/electric/v1/electric.proto"


export class UpdateDemandRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateDemandRequest;

  getDemand(): smartcore_bos_electric_v1_electric_pb.ElectricDemand | undefined;
  setDemand(value?: smartcore_bos_electric_v1_electric_pb.ElectricDemand): UpdateDemandRequest;
  hasDemand(): boolean;
  clearDemand(): UpdateDemandRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateDemandRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateDemandRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateDemandRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateDemandRequest): UpdateDemandRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateDemandRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateDemandRequest;
  static deserializeBinaryFromReader(message: UpdateDemandRequest, reader: jspb.BinaryReader): UpdateDemandRequest;
}

export namespace UpdateDemandRequest {
  export type AsObject = {
    name: string;
    demand?: smartcore_bos_electric_v1_electric_pb.ElectricDemand.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class CreateModeRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CreateModeRequest;

  getMode(): smartcore_bos_electric_v1_electric_pb.ElectricMode | undefined;
  setMode(value?: smartcore_bos_electric_v1_electric_pb.ElectricMode): CreateModeRequest;
  hasMode(): boolean;
  clearMode(): CreateModeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateModeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateModeRequest): CreateModeRequest.AsObject;
  static serializeBinaryToWriter(message: CreateModeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateModeRequest;
  static deserializeBinaryFromReader(message: CreateModeRequest, reader: jspb.BinaryReader): CreateModeRequest;
}

export namespace CreateModeRequest {
  export type AsObject = {
    name: string;
    mode?: smartcore_bos_electric_v1_electric_pb.ElectricMode.AsObject;
  };
}

export class UpdateModeRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateModeRequest;

  getMode(): smartcore_bos_electric_v1_electric_pb.ElectricMode | undefined;
  setMode(value?: smartcore_bos_electric_v1_electric_pb.ElectricMode): UpdateModeRequest;
  hasMode(): boolean;
  clearMode(): UpdateModeRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateModeRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateModeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateModeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateModeRequest): UpdateModeRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateModeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateModeRequest;
  static deserializeBinaryFromReader(message: UpdateModeRequest, reader: jspb.BinaryReader): UpdateModeRequest;
}

export namespace UpdateModeRequest {
  export type AsObject = {
    name: string;
    mode?: smartcore_bos_electric_v1_electric_pb.ElectricMode.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class DeleteModeRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DeleteModeRequest;

  getId(): string;
  setId(value: string): DeleteModeRequest;

  getAllowMissing(): boolean;
  setAllowMissing(value: boolean): DeleteModeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteModeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteModeRequest): DeleteModeRequest.AsObject;
  static serializeBinaryToWriter(message: DeleteModeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteModeRequest;
  static deserializeBinaryFromReader(message: DeleteModeRequest, reader: jspb.BinaryReader): DeleteModeRequest;
}

export namespace DeleteModeRequest {
  export type AsObject = {
    name: string;
    id: string;
    allowMissing: boolean;
  };
}

