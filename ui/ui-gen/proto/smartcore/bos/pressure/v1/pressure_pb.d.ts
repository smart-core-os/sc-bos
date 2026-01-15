import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '@smart-core-os/sc-api-grpc-web/types/info_pb'; // proto import: "types/info.proto"


export class Pressure extends jspb.Message {
  getTargetPressure(): number;
  setTargetPressure(value: number): Pressure;
  hasTargetPressure(): boolean;
  clearTargetPressure(): Pressure;

  getPressure(): number;
  setPressure(value: number): Pressure;
  hasPressure(): boolean;
  clearPressure(): Pressure;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Pressure.AsObject;
  static toObject(includeInstance: boolean, msg: Pressure): Pressure.AsObject;
  static serializeBinaryToWriter(message: Pressure, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Pressure;
  static deserializeBinaryFromReader(message: Pressure, reader: jspb.BinaryReader): Pressure;
}

export namespace Pressure {
  export type AsObject = {
    targetPressure?: number;
    pressure?: number;
  };

  export enum TargetPressureCase {
    _TARGET_PRESSURE_NOT_SET = 0,
    TARGET_PRESSURE = 1,
  }

  export enum PressureCase {
    _PRESSURE_NOT_SET = 0,
    PRESSURE = 2,
  }
}

export class GetPressureRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetPressureRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetPressureRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetPressureRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetPressureRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetPressureRequest): GetPressureRequest.AsObject;
  static serializeBinaryToWriter(message: GetPressureRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetPressureRequest;
  static deserializeBinaryFromReader(message: GetPressureRequest, reader: jspb.BinaryReader): GetPressureRequest;
}

export namespace GetPressureRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullPressureRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullPressureRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullPressureRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullPressureRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullPressureRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullPressureRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullPressureRequest): PullPressureRequest.AsObject;
  static serializeBinaryToWriter(message: PullPressureRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullPressureRequest;
  static deserializeBinaryFromReader(message: PullPressureRequest, reader: jspb.BinaryReader): PullPressureRequest;
}

export namespace PullPressureRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullPressureResponse extends jspb.Message {
  getChangesList(): Array<PullPressureResponse.Change>;
  setChangesList(value: Array<PullPressureResponse.Change>): PullPressureResponse;
  clearChangesList(): PullPressureResponse;
  addChanges(value?: PullPressureResponse.Change, index?: number): PullPressureResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullPressureResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullPressureResponse): PullPressureResponse.AsObject;
  static serializeBinaryToWriter(message: PullPressureResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullPressureResponse;
  static deserializeBinaryFromReader(message: PullPressureResponse, reader: jspb.BinaryReader): PullPressureResponse;
}

export namespace PullPressureResponse {
  export type AsObject = {
    changesList: Array<PullPressureResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getPressure(): Pressure | undefined;
    setPressure(value?: Pressure): Change;
    hasPressure(): boolean;
    clearPressure(): Change;

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
      pressure?: Pressure.AsObject;
    };
  }

}

export class UpdatePressureRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdatePressureRequest;

  getPressure(): Pressure | undefined;
  setPressure(value?: Pressure): UpdatePressureRequest;
  hasPressure(): boolean;
  clearPressure(): UpdatePressureRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdatePressureRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdatePressureRequest;

  getDelta(): boolean;
  setDelta(value: boolean): UpdatePressureRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdatePressureRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdatePressureRequest): UpdatePressureRequest.AsObject;
  static serializeBinaryToWriter(message: UpdatePressureRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdatePressureRequest;
  static deserializeBinaryFromReader(message: UpdatePressureRequest, reader: jspb.BinaryReader): UpdatePressureRequest;
}

export namespace UpdatePressureRequest {
  export type AsObject = {
    name: string;
    pressure?: Pressure.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    delta: boolean;
  };
}

export class DescribePressureRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribePressureRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribePressureRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribePressureRequest): DescribePressureRequest.AsObject;
  static serializeBinaryToWriter(message: DescribePressureRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribePressureRequest;
  static deserializeBinaryFromReader(message: DescribePressureRequest, reader: jspb.BinaryReader): DescribePressureRequest;
}

export namespace DescribePressureRequest {
  export type AsObject = {
    name: string;
  };
}

export class PressureSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): PressureSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): PressureSupport;

  getPressureUnit(): string;
  setPressureUnit(value: string): PressureSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PressureSupport.AsObject;
  static toObject(includeInstance: boolean, msg: PressureSupport): PressureSupport.AsObject;
  static serializeBinaryToWriter(message: PressureSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PressureSupport;
  static deserializeBinaryFromReader(message: PressureSupport, reader: jspb.BinaryReader): PressureSupport;
}

export namespace PressureSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    pressureUnit: string;
  };
}

