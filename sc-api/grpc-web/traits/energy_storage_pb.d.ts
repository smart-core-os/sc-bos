import * as jspb from 'google-protobuf'

import * as google_protobuf_duration_pb from 'google-protobuf/google/protobuf/duration_pb'; // proto import: "google/protobuf/duration.proto"
import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"


export class EnergyLevelSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): EnergyLevelSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): EnergyLevelSupport;

  getRechargeable(): boolean;
  setRechargeable(value: boolean): EnergyLevelSupport;

  getChargeControl(): EnergyLevelSupport.ChargeControl;
  setChargeControl(value: EnergyLevelSupport.ChargeControl): EnergyLevelSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): EnergyLevelSupport.AsObject;
  static toObject(includeInstance: boolean, msg: EnergyLevelSupport): EnergyLevelSupport.AsObject;
  static serializeBinaryToWriter(message: EnergyLevelSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): EnergyLevelSupport;
  static deserializeBinaryFromReader(message: EnergyLevelSupport, reader: jspb.BinaryReader): EnergyLevelSupport;
}

export namespace EnergyLevelSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    rechargeable: boolean;
    chargeControl: EnergyLevelSupport.ChargeControl;
  };

  export enum ChargeControl {
    CHARGE_CONTROL_UNSPECIFIED = 0,
    NONE = 1,
    DEVICE = 2,
    EXTERNAL = 3,
    ALL = 4,
  }
}

export class EnergyLevel extends jspb.Message {
  getDischarge(): EnergyLevel.Transfer | undefined;
  setDischarge(value?: EnergyLevel.Transfer): EnergyLevel;
  hasDischarge(): boolean;
  clearDischarge(): EnergyLevel;

  getCharge(): EnergyLevel.Transfer | undefined;
  setCharge(value?: EnergyLevel.Transfer): EnergyLevel;
  hasCharge(): boolean;
  clearCharge(): EnergyLevel;

  getIdle(): EnergyLevel.Steady | undefined;
  setIdle(value?: EnergyLevel.Steady): EnergyLevel;
  hasIdle(): boolean;
  clearIdle(): EnergyLevel;

  getQuantity(): EnergyLevel.Quantity | undefined;
  setQuantity(value?: EnergyLevel.Quantity): EnergyLevel;
  hasQuantity(): boolean;
  clearQuantity(): EnergyLevel;

  getPluggedIn(): boolean;
  setPluggedIn(value: boolean): EnergyLevel;

  getFlowCase(): EnergyLevel.FlowCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): EnergyLevel.AsObject;
  static toObject(includeInstance: boolean, msg: EnergyLevel): EnergyLevel.AsObject;
  static serializeBinaryToWriter(message: EnergyLevel, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): EnergyLevel;
  static deserializeBinaryFromReader(message: EnergyLevel, reader: jspb.BinaryReader): EnergyLevel;
}

export namespace EnergyLevel {
  export type AsObject = {
    discharge?: EnergyLevel.Transfer.AsObject;
    charge?: EnergyLevel.Transfer.AsObject;
    idle?: EnergyLevel.Steady.AsObject;
    quantity?: EnergyLevel.Quantity.AsObject;
    pluggedIn: boolean;
  };

  export class Transfer extends jspb.Message {
    getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): Transfer;
    hasStartTime(): boolean;
    clearStartTime(): Transfer;

    getTime(): google_protobuf_duration_pb.Duration | undefined;
    setTime(value?: google_protobuf_duration_pb.Duration): Transfer;
    hasTime(): boolean;
    clearTime(): Transfer;

    getDistanceKm(): number;
    setDistanceKm(value: number): Transfer;

    getSpeed(): EnergyLevel.Transfer.Speed;
    setSpeed(value: EnergyLevel.Transfer.Speed): Transfer;

    getTarget(): EnergyLevel.Quantity | undefined;
    setTarget(value?: EnergyLevel.Quantity): Transfer;
    hasTarget(): boolean;
    clearTarget(): Transfer;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Transfer.AsObject;
    static toObject(includeInstance: boolean, msg: Transfer): Transfer.AsObject;
    static serializeBinaryToWriter(message: Transfer, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Transfer;
    static deserializeBinaryFromReader(message: Transfer, reader: jspb.BinaryReader): Transfer;
  }

  export namespace Transfer {
    export type AsObject = {
      startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      time?: google_protobuf_duration_pb.Duration.AsObject;
      distanceKm: number;
      speed: EnergyLevel.Transfer.Speed;
      target?: EnergyLevel.Quantity.AsObject;
    };

    export enum Speed {
      SPEED_UNSPECIFIED = 0,
      EXTRA_SLOW = 1,
      SLOW = 2,
      NORMAL = 3,
      FAST = 4,
      EXTRA_FAST = 5,
    }
  }


  export class Steady extends jspb.Message {
    getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): Steady;
    hasStartTime(): boolean;
    clearStartTime(): Steady;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Steady.AsObject;
    static toObject(includeInstance: boolean, msg: Steady): Steady.AsObject;
    static serializeBinaryToWriter(message: Steady, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Steady;
    static deserializeBinaryFromReader(message: Steady, reader: jspb.BinaryReader): Steady;
  }

  export namespace Steady {
    export type AsObject = {
      startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };
  }


  export class Quantity extends jspb.Message {
    getPercentage(): number;
    setPercentage(value: number): Quantity;

    getEnergyKwh(): number;
    setEnergyKwh(value: number): Quantity;

    getDescriptive(): EnergyLevel.Quantity.Threshold;
    setDescriptive(value: EnergyLevel.Quantity.Threshold): Quantity;

    getDistanceKm(): number;
    setDistanceKm(value: number): Quantity;

    getVoltage(): number;
    setVoltage(value: number): Quantity;
    hasVoltage(): boolean;
    clearVoltage(): Quantity;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Quantity.AsObject;
    static toObject(includeInstance: boolean, msg: Quantity): Quantity.AsObject;
    static serializeBinaryToWriter(message: Quantity, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Quantity;
    static deserializeBinaryFromReader(message: Quantity, reader: jspb.BinaryReader): Quantity;
  }

  export namespace Quantity {
    export type AsObject = {
      percentage: number;
      energyKwh: number;
      descriptive: EnergyLevel.Quantity.Threshold;
      distanceKm: number;
      voltage?: number;
    };

    export enum Threshold {
      THRESHOLD_UNSPECIFIED = 0,
      CRITICALLY_LOW = 1,
      EMPTY = 2,
      LOW = 3,
      MEDIUM = 4,
      HIGH = 5,
      FULL = 7,
      CRITICALLY_HIGH = 8,
    }

    export enum VoltageCase {
      _VOLTAGE_NOT_SET = 0,
      VOLTAGE = 5,
    }
  }


  export enum FlowCase {
    FLOW_NOT_SET = 0,
    DISCHARGE = 2,
    CHARGE = 3,
    IDLE = 4,
  }
}

export class GetEnergyLevelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetEnergyLevelRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetEnergyLevelRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetEnergyLevelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEnergyLevelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetEnergyLevelRequest): GetEnergyLevelRequest.AsObject;
  static serializeBinaryToWriter(message: GetEnergyLevelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetEnergyLevelRequest;
  static deserializeBinaryFromReader(message: GetEnergyLevelRequest, reader: jspb.BinaryReader): GetEnergyLevelRequest;
}

export namespace GetEnergyLevelRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullEnergyLevelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullEnergyLevelRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullEnergyLevelRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullEnergyLevelRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullEnergyLevelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullEnergyLevelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullEnergyLevelRequest): PullEnergyLevelRequest.AsObject;
  static serializeBinaryToWriter(message: PullEnergyLevelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullEnergyLevelRequest;
  static deserializeBinaryFromReader(message: PullEnergyLevelRequest, reader: jspb.BinaryReader): PullEnergyLevelRequest;
}

export namespace PullEnergyLevelRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullEnergyLevelResponse extends jspb.Message {
  getChangesList(): Array<PullEnergyLevelResponse.Change>;
  setChangesList(value: Array<PullEnergyLevelResponse.Change>): PullEnergyLevelResponse;
  clearChangesList(): PullEnergyLevelResponse;
  addChanges(value?: PullEnergyLevelResponse.Change, index?: number): PullEnergyLevelResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullEnergyLevelResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullEnergyLevelResponse): PullEnergyLevelResponse.AsObject;
  static serializeBinaryToWriter(message: PullEnergyLevelResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullEnergyLevelResponse;
  static deserializeBinaryFromReader(message: PullEnergyLevelResponse, reader: jspb.BinaryReader): PullEnergyLevelResponse;
}

export namespace PullEnergyLevelResponse {
  export type AsObject = {
    changesList: Array<PullEnergyLevelResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getEnergyLevel(): EnergyLevel | undefined;
    setEnergyLevel(value?: EnergyLevel): Change;
    hasEnergyLevel(): boolean;
    clearEnergyLevel(): Change;

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
      energyLevel?: EnergyLevel.AsObject;
    };
  }

}

export class ChargeRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ChargeRequest;

  getCharge(): boolean;
  setCharge(value: boolean): ChargeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ChargeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ChargeRequest): ChargeRequest.AsObject;
  static serializeBinaryToWriter(message: ChargeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ChargeRequest;
  static deserializeBinaryFromReader(message: ChargeRequest, reader: jspb.BinaryReader): ChargeRequest;
}

export namespace ChargeRequest {
  export type AsObject = {
    name: string;
    charge: boolean;
  };
}

export class ChargeResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ChargeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ChargeResponse): ChargeResponse.AsObject;
  static serializeBinaryToWriter(message: ChargeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ChargeResponse;
  static deserializeBinaryFromReader(message: ChargeResponse, reader: jspb.BinaryReader): ChargeResponse;
}

export namespace ChargeResponse {
  export type AsObject = {
  };
}

export class DescribeEnergyLevelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeEnergyLevelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeEnergyLevelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeEnergyLevelRequest): DescribeEnergyLevelRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeEnergyLevelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeEnergyLevelRequest;
  static deserializeBinaryFromReader(message: DescribeEnergyLevelRequest, reader: jspb.BinaryReader): DescribeEnergyLevelRequest;
}

export namespace DescribeEnergyLevelRequest {
  export type AsObject = {
    name: string;
  };
}

