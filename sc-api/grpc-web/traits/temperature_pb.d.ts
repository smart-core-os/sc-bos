import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_unit_pb from '../types/unit_pb'; // proto import: "types/unit.proto"


export class Temperature extends jspb.Message {
  getSetPoint(): types_unit_pb.Temperature | undefined;
  setSetPoint(value?: types_unit_pb.Temperature): Temperature;
  hasSetPoint(): boolean;
  clearSetPoint(): Temperature;

  getMeasured(): types_unit_pb.Temperature | undefined;
  setMeasured(value?: types_unit_pb.Temperature): Temperature;
  hasMeasured(): boolean;
  clearMeasured(): Temperature;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Temperature.AsObject;
  static toObject(includeInstance: boolean, msg: Temperature): Temperature.AsObject;
  static serializeBinaryToWriter(message: Temperature, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Temperature;
  static deserializeBinaryFromReader(message: Temperature, reader: jspb.BinaryReader): Temperature;
}

export namespace Temperature {
  export type AsObject = {
    setPoint?: types_unit_pb.Temperature.AsObject;
    measured?: types_unit_pb.Temperature.AsObject;
  };
}

export class GetTemperatureRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetTemperatureRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetTemperatureRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetTemperatureRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetTemperatureRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetTemperatureRequest): GetTemperatureRequest.AsObject;
  static serializeBinaryToWriter(message: GetTemperatureRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetTemperatureRequest;
  static deserializeBinaryFromReader(message: GetTemperatureRequest, reader: jspb.BinaryReader): GetTemperatureRequest;
}

export namespace GetTemperatureRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullTemperatureRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullTemperatureRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullTemperatureRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullTemperatureRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullTemperatureRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullTemperatureRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullTemperatureRequest): PullTemperatureRequest.AsObject;
  static serializeBinaryToWriter(message: PullTemperatureRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullTemperatureRequest;
  static deserializeBinaryFromReader(message: PullTemperatureRequest, reader: jspb.BinaryReader): PullTemperatureRequest;
}

export namespace PullTemperatureRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullTemperatureResponse extends jspb.Message {
  getChangesList(): Array<PullTemperatureResponse.Change>;
  setChangesList(value: Array<PullTemperatureResponse.Change>): PullTemperatureResponse;
  clearChangesList(): PullTemperatureResponse;
  addChanges(value?: PullTemperatureResponse.Change, index?: number): PullTemperatureResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullTemperatureResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullTemperatureResponse): PullTemperatureResponse.AsObject;
  static serializeBinaryToWriter(message: PullTemperatureResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullTemperatureResponse;
  static deserializeBinaryFromReader(message: PullTemperatureResponse, reader: jspb.BinaryReader): PullTemperatureResponse;
}

export namespace PullTemperatureResponse {
  export type AsObject = {
    changesList: Array<PullTemperatureResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getTemperature(): Temperature | undefined;
    setTemperature(value?: Temperature): Change;
    hasTemperature(): boolean;
    clearTemperature(): Change;

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
      temperature?: Temperature.AsObject;
    };
  }

}

export class UpdateTemperatureRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateTemperatureRequest;

  getTemperature(): Temperature | undefined;
  setTemperature(value?: Temperature): UpdateTemperatureRequest;
  hasTemperature(): boolean;
  clearTemperature(): UpdateTemperatureRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateTemperatureRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateTemperatureRequest;

  getDelta(): boolean;
  setDelta(value: boolean): UpdateTemperatureRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateTemperatureRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateTemperatureRequest): UpdateTemperatureRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateTemperatureRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateTemperatureRequest;
  static deserializeBinaryFromReader(message: UpdateTemperatureRequest, reader: jspb.BinaryReader): UpdateTemperatureRequest;
}

export namespace UpdateTemperatureRequest {
  export type AsObject = {
    name: string;
    temperature?: Temperature.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    delta: boolean;
  };
}

