import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"
import * as types_unit_pb from '../types/unit_pb'; // proto import: "types/unit.proto"


export class AirTemperature extends jspb.Message {
  getMode(): AirTemperature.Mode;
  setMode(value: AirTemperature.Mode): AirTemperature;

  getTemperatureSetPoint(): types_unit_pb.Temperature | undefined;
  setTemperatureSetPoint(value?: types_unit_pb.Temperature): AirTemperature;
  hasTemperatureSetPoint(): boolean;
  clearTemperatureSetPoint(): AirTemperature;

  getTemperatureSetPointDelta(): types_unit_pb.Temperature | undefined;
  setTemperatureSetPointDelta(value?: types_unit_pb.Temperature): AirTemperature;
  hasTemperatureSetPointDelta(): boolean;
  clearTemperatureSetPointDelta(): AirTemperature;

  getTemperatureRange(): TemperatureRange | undefined;
  setTemperatureRange(value?: TemperatureRange): AirTemperature;
  hasTemperatureRange(): boolean;
  clearTemperatureRange(): AirTemperature;

  getAmbientTemperature(): types_unit_pb.Temperature | undefined;
  setAmbientTemperature(value?: types_unit_pb.Temperature): AirTemperature;
  hasAmbientTemperature(): boolean;
  clearAmbientTemperature(): AirTemperature;

  getAmbientHumidity(): number;
  setAmbientHumidity(value: number): AirTemperature;
  hasAmbientHumidity(): boolean;
  clearAmbientHumidity(): AirTemperature;

  getDewPoint(): types_unit_pb.Temperature | undefined;
  setDewPoint(value?: types_unit_pb.Temperature): AirTemperature;
  hasDewPoint(): boolean;
  clearDewPoint(): AirTemperature;

  getTemperatureGoalCase(): AirTemperature.TemperatureGoalCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AirTemperature.AsObject;
  static toObject(includeInstance: boolean, msg: AirTemperature): AirTemperature.AsObject;
  static serializeBinaryToWriter(message: AirTemperature, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AirTemperature;
  static deserializeBinaryFromReader(message: AirTemperature, reader: jspb.BinaryReader): AirTemperature;
}

export namespace AirTemperature {
  export type AsObject = {
    mode: AirTemperature.Mode;
    temperatureSetPoint?: types_unit_pb.Temperature.AsObject;
    temperatureSetPointDelta?: types_unit_pb.Temperature.AsObject;
    temperatureRange?: TemperatureRange.AsObject;
    ambientTemperature?: types_unit_pb.Temperature.AsObject;
    ambientHumidity?: number;
    dewPoint?: types_unit_pb.Temperature.AsObject;
  };

  export enum Mode {
    MODE_UNSPECIFIED = 0,
    ON = 1,
    OFF = 2,
    HEAT = 3,
    COOL = 4,
    HEAT_COOL = 5,
    AUTO = 6,
    FAN_ONLY = 7,
    ECO = 8,
    PURIFIER = 9,
    DRY = 10,
    LOCKED = 11,
  }

  export enum TemperatureGoalCase {
    TEMPERATURE_GOAL_NOT_SET = 0,
    TEMPERATURE_SET_POINT = 2,
    TEMPERATURE_SET_POINT_DELTA = 3,
    TEMPERATURE_RANGE = 4,
  }

  export enum AmbientHumidityCase {
    _AMBIENT_HUMIDITY_NOT_SET = 0,
    AMBIENT_HUMIDITY = 6,
  }
}

export class TemperatureRange extends jspb.Message {
  getLow(): types_unit_pb.Temperature | undefined;
  setLow(value?: types_unit_pb.Temperature): TemperatureRange;
  hasLow(): boolean;
  clearLow(): TemperatureRange;

  getHigh(): types_unit_pb.Temperature | undefined;
  setHigh(value?: types_unit_pb.Temperature): TemperatureRange;
  hasHigh(): boolean;
  clearHigh(): TemperatureRange;

  getIdeal(): types_unit_pb.Temperature | undefined;
  setIdeal(value?: types_unit_pb.Temperature): TemperatureRange;
  hasIdeal(): boolean;
  clearIdeal(): TemperatureRange;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TemperatureRange.AsObject;
  static toObject(includeInstance: boolean, msg: TemperatureRange): TemperatureRange.AsObject;
  static serializeBinaryToWriter(message: TemperatureRange, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TemperatureRange;
  static deserializeBinaryFromReader(message: TemperatureRange, reader: jspb.BinaryReader): TemperatureRange;
}

export namespace TemperatureRange {
  export type AsObject = {
    low?: types_unit_pb.Temperature.AsObject;
    high?: types_unit_pb.Temperature.AsObject;
    ideal?: types_unit_pb.Temperature.AsObject;
  };
}

export class AirTemperatureSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): AirTemperatureSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): AirTemperatureSupport;

  getNativeUnit(): types_unit_pb.TemperatureUnit;
  setNativeUnit(value: types_unit_pb.TemperatureUnit): AirTemperatureSupport;

  getSupportedModesList(): Array<AirTemperature.Mode>;
  setSupportedModesList(value: Array<AirTemperature.Mode>): AirTemperatureSupport;
  clearSupportedModesList(): AirTemperatureSupport;
  addSupportedModes(value: AirTemperature.Mode, index?: number): AirTemperatureSupport;

  getMinRangeCelsius(): number;
  setMinRangeCelsius(value: number): AirTemperatureSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AirTemperatureSupport.AsObject;
  static toObject(includeInstance: boolean, msg: AirTemperatureSupport): AirTemperatureSupport.AsObject;
  static serializeBinaryToWriter(message: AirTemperatureSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AirTemperatureSupport;
  static deserializeBinaryFromReader(message: AirTemperatureSupport, reader: jspb.BinaryReader): AirTemperatureSupport;
}

export namespace AirTemperatureSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    nativeUnit: types_unit_pb.TemperatureUnit;
    supportedModesList: Array<AirTemperature.Mode>;
    minRangeCelsius: number;
  };
}

export class GetAirTemperatureRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetAirTemperatureRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetAirTemperatureRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetAirTemperatureRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetAirTemperatureRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetAirTemperatureRequest): GetAirTemperatureRequest.AsObject;
  static serializeBinaryToWriter(message: GetAirTemperatureRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetAirTemperatureRequest;
  static deserializeBinaryFromReader(message: GetAirTemperatureRequest, reader: jspb.BinaryReader): GetAirTemperatureRequest;
}

export namespace GetAirTemperatureRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateAirTemperatureRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateAirTemperatureRequest;

  getState(): AirTemperature | undefined;
  setState(value?: AirTemperature): UpdateAirTemperatureRequest;
  hasState(): boolean;
  clearState(): UpdateAirTemperatureRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateAirTemperatureRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateAirTemperatureRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateAirTemperatureRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateAirTemperatureRequest): UpdateAirTemperatureRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateAirTemperatureRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateAirTemperatureRequest;
  static deserializeBinaryFromReader(message: UpdateAirTemperatureRequest, reader: jspb.BinaryReader): UpdateAirTemperatureRequest;
}

export namespace UpdateAirTemperatureRequest {
  export type AsObject = {
    name: string;
    state?: AirTemperature.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullAirTemperatureRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullAirTemperatureRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullAirTemperatureRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullAirTemperatureRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullAirTemperatureRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullAirTemperatureRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullAirTemperatureRequest): PullAirTemperatureRequest.AsObject;
  static serializeBinaryToWriter(message: PullAirTemperatureRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullAirTemperatureRequest;
  static deserializeBinaryFromReader(message: PullAirTemperatureRequest, reader: jspb.BinaryReader): PullAirTemperatureRequest;
}

export namespace PullAirTemperatureRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullAirTemperatureResponse extends jspb.Message {
  getChangesList(): Array<PullAirTemperatureResponse.Change>;
  setChangesList(value: Array<PullAirTemperatureResponse.Change>): PullAirTemperatureResponse;
  clearChangesList(): PullAirTemperatureResponse;
  addChanges(value?: PullAirTemperatureResponse.Change, index?: number): PullAirTemperatureResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullAirTemperatureResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullAirTemperatureResponse): PullAirTemperatureResponse.AsObject;
  static serializeBinaryToWriter(message: PullAirTemperatureResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullAirTemperatureResponse;
  static deserializeBinaryFromReader(message: PullAirTemperatureResponse, reader: jspb.BinaryReader): PullAirTemperatureResponse;
}

export namespace PullAirTemperatureResponse {
  export type AsObject = {
    changesList: Array<PullAirTemperatureResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getAirTemperature(): AirTemperature | undefined;
    setAirTemperature(value?: AirTemperature): Change;
    hasAirTemperature(): boolean;
    clearAirTemperature(): Change;

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
      airTemperature?: AirTemperature.AsObject;
    };
  }

}

export class DescribeAirTemperatureRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeAirTemperatureRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeAirTemperatureRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeAirTemperatureRequest): DescribeAirTemperatureRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeAirTemperatureRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeAirTemperatureRequest;
  static deserializeBinaryFromReader(message: DescribeAirTemperatureRequest, reader: jspb.BinaryReader): DescribeAirTemperatureRequest;
}

export namespace DescribeAirTemperatureRequest {
  export type AsObject = {
    name: string;
  };
}

