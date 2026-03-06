import * as jspb from 'google-protobuf'

import * as google_protobuf_duration_pb from 'google-protobuf/google/protobuf/duration_pb'; // proto import: "google/protobuf/duration.proto"
import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_change_pb from '../types/change_pb'; // proto import: "types/change.proto"


export class ElectricDemand extends jspb.Message {
  getCurrent(): number;
  setCurrent(value: number): ElectricDemand;

  getVoltage(): number;
  setVoltage(value: number): ElectricDemand;
  hasVoltage(): boolean;
  clearVoltage(): ElectricDemand;

  getRating(): number;
  setRating(value: number): ElectricDemand;

  getPowerFactor(): number;
  setPowerFactor(value: number): ElectricDemand;
  hasPowerFactor(): boolean;
  clearPowerFactor(): ElectricDemand;

  getRealPower(): number;
  setRealPower(value: number): ElectricDemand;
  hasRealPower(): boolean;
  clearRealPower(): ElectricDemand;

  getApparentPower(): number;
  setApparentPower(value: number): ElectricDemand;
  hasApparentPower(): boolean;
  clearApparentPower(): ElectricDemand;

  getReactivePower(): number;
  setReactivePower(value: number): ElectricDemand;
  hasReactivePower(): boolean;
  clearReactivePower(): ElectricDemand;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ElectricDemand.AsObject;
  static toObject(includeInstance: boolean, msg: ElectricDemand): ElectricDemand.AsObject;
  static serializeBinaryToWriter(message: ElectricDemand, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ElectricDemand;
  static deserializeBinaryFromReader(message: ElectricDemand, reader: jspb.BinaryReader): ElectricDemand;
}

export namespace ElectricDemand {
  export type AsObject = {
    current: number;
    voltage?: number;
    rating: number;
    powerFactor?: number;
    realPower?: number;
    apparentPower?: number;
    reactivePower?: number;
  };

  export enum VoltageCase {
    _VOLTAGE_NOT_SET = 0,
    VOLTAGE = 2,
  }

  export enum PowerFactorCase {
    _POWER_FACTOR_NOT_SET = 0,
    POWER_FACTOR = 4,
  }

  export enum RealPowerCase {
    _REAL_POWER_NOT_SET = 0,
    REAL_POWER = 5,
  }

  export enum ApparentPowerCase {
    _APPARENT_POWER_NOT_SET = 0,
    APPARENT_POWER = 6,
  }

  export enum ReactivePowerCase {
    _REACTIVE_POWER_NOT_SET = 0,
    REACTIVE_POWER = 7,
  }
}

export class ElectricMode extends jspb.Message {
  getId(): string;
  setId(value: string): ElectricMode;

  getTitle(): string;
  setTitle(value: string): ElectricMode;

  getDescription(): string;
  setDescription(value: string): ElectricMode;

  getVoltage(): number;
  setVoltage(value: number): ElectricMode;

  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): ElectricMode;
  hasStartTime(): boolean;
  clearStartTime(): ElectricMode;

  getSegmentsList(): Array<ElectricMode.Segment>;
  setSegmentsList(value: Array<ElectricMode.Segment>): ElectricMode;
  clearSegmentsList(): ElectricMode;
  addSegments(value?: ElectricMode.Segment, index?: number): ElectricMode.Segment;

  getNormal(): boolean;
  setNormal(value: boolean): ElectricMode;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ElectricMode.AsObject;
  static toObject(includeInstance: boolean, msg: ElectricMode): ElectricMode.AsObject;
  static serializeBinaryToWriter(message: ElectricMode, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ElectricMode;
  static deserializeBinaryFromReader(message: ElectricMode, reader: jspb.BinaryReader): ElectricMode;
}

export namespace ElectricMode {
  export type AsObject = {
    id: string;
    title: string;
    description: string;
    voltage: number;
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    segmentsList: Array<ElectricMode.Segment.AsObject>;
    normal: boolean;
  };

  export class Segment extends jspb.Message {
    getLength(): google_protobuf_duration_pb.Duration | undefined;
    setLength(value?: google_protobuf_duration_pb.Duration): Segment;
    hasLength(): boolean;
    clearLength(): Segment;

    getMagnitude(): number;
    setMagnitude(value: number): Segment;

    getFixed(): number;
    setFixed(value: number): Segment;
    hasFixed(): boolean;
    clearFixed(): Segment;

    getShapeCase(): Segment.ShapeCase;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Segment.AsObject;
    static toObject(includeInstance: boolean, msg: Segment): Segment.AsObject;
    static serializeBinaryToWriter(message: Segment, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Segment;
    static deserializeBinaryFromReader(message: Segment, reader: jspb.BinaryReader): Segment;
  }

  export namespace Segment {
    export type AsObject = {
      length?: google_protobuf_duration_pb.Duration.AsObject;
      magnitude: number;
      fixed?: number;
    };

    export enum ShapeCase {
      SHAPE_NOT_SET = 0,
      FIXED = 3,
    }
  }

}

export class GetDemandRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetDemandRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetDemandRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetDemandRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDemandRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetDemandRequest): GetDemandRequest.AsObject;
  static serializeBinaryToWriter(message: GetDemandRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDemandRequest;
  static deserializeBinaryFromReader(message: GetDemandRequest, reader: jspb.BinaryReader): GetDemandRequest;
}

export namespace GetDemandRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullDemandRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullDemandRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullDemandRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullDemandRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullDemandRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullDemandRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullDemandRequest): PullDemandRequest.AsObject;
  static serializeBinaryToWriter(message: PullDemandRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullDemandRequest;
  static deserializeBinaryFromReader(message: PullDemandRequest, reader: jspb.BinaryReader): PullDemandRequest;
}

export namespace PullDemandRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullDemandResponse extends jspb.Message {
  getChangesList(): Array<PullDemandResponse.Change>;
  setChangesList(value: Array<PullDemandResponse.Change>): PullDemandResponse;
  clearChangesList(): PullDemandResponse;
  addChanges(value?: PullDemandResponse.Change, index?: number): PullDemandResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullDemandResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullDemandResponse): PullDemandResponse.AsObject;
  static serializeBinaryToWriter(message: PullDemandResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullDemandResponse;
  static deserializeBinaryFromReader(message: PullDemandResponse, reader: jspb.BinaryReader): PullDemandResponse;
}

export namespace PullDemandResponse {
  export type AsObject = {
    changesList: Array<PullDemandResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getDemand(): ElectricDemand | undefined;
    setDemand(value?: ElectricDemand): Change;
    hasDemand(): boolean;
    clearDemand(): Change;

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
      demand?: ElectricDemand.AsObject;
    };
  }

}

export class GetActiveModeRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetActiveModeRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetActiveModeRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetActiveModeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetActiveModeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetActiveModeRequest): GetActiveModeRequest.AsObject;
  static serializeBinaryToWriter(message: GetActiveModeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetActiveModeRequest;
  static deserializeBinaryFromReader(message: GetActiveModeRequest, reader: jspb.BinaryReader): GetActiveModeRequest;
}

export namespace GetActiveModeRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateActiveModeRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateActiveModeRequest;

  getActiveMode(): ElectricMode | undefined;
  setActiveMode(value?: ElectricMode): UpdateActiveModeRequest;
  hasActiveMode(): boolean;
  clearActiveMode(): UpdateActiveModeRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateActiveModeRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateActiveModeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateActiveModeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateActiveModeRequest): UpdateActiveModeRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateActiveModeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateActiveModeRequest;
  static deserializeBinaryFromReader(message: UpdateActiveModeRequest, reader: jspb.BinaryReader): UpdateActiveModeRequest;
}

export namespace UpdateActiveModeRequest {
  export type AsObject = {
    name: string;
    activeMode?: ElectricMode.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class ClearActiveModeRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ClearActiveModeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ClearActiveModeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ClearActiveModeRequest): ClearActiveModeRequest.AsObject;
  static serializeBinaryToWriter(message: ClearActiveModeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ClearActiveModeRequest;
  static deserializeBinaryFromReader(message: ClearActiveModeRequest, reader: jspb.BinaryReader): ClearActiveModeRequest;
}

export namespace ClearActiveModeRequest {
  export type AsObject = {
    name: string;
  };
}

export class PullActiveModeRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullActiveModeRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullActiveModeRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullActiveModeRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullActiveModeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullActiveModeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullActiveModeRequest): PullActiveModeRequest.AsObject;
  static serializeBinaryToWriter(message: PullActiveModeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullActiveModeRequest;
  static deserializeBinaryFromReader(message: PullActiveModeRequest, reader: jspb.BinaryReader): PullActiveModeRequest;
}

export namespace PullActiveModeRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullActiveModeResponse extends jspb.Message {
  getChangesList(): Array<PullActiveModeResponse.Change>;
  setChangesList(value: Array<PullActiveModeResponse.Change>): PullActiveModeResponse;
  clearChangesList(): PullActiveModeResponse;
  addChanges(value?: PullActiveModeResponse.Change, index?: number): PullActiveModeResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullActiveModeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullActiveModeResponse): PullActiveModeResponse.AsObject;
  static serializeBinaryToWriter(message: PullActiveModeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullActiveModeResponse;
  static deserializeBinaryFromReader(message: PullActiveModeResponse, reader: jspb.BinaryReader): PullActiveModeResponse;
}

export namespace PullActiveModeResponse {
  export type AsObject = {
    changesList: Array<PullActiveModeResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getActiveMode(): ElectricMode | undefined;
    setActiveMode(value?: ElectricMode): Change;
    hasActiveMode(): boolean;
    clearActiveMode(): Change;

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
      activeMode?: ElectricMode.AsObject;
    };
  }

}

export class ListModesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListModesRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListModesRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListModesRequest;

  getPageSize(): number;
  setPageSize(value: number): ListModesRequest;

  getPageToken(): string;
  setPageToken(value: string): ListModesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListModesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListModesRequest): ListModesRequest.AsObject;
  static serializeBinaryToWriter(message: ListModesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListModesRequest;
  static deserializeBinaryFromReader(message: ListModesRequest, reader: jspb.BinaryReader): ListModesRequest;
}

export namespace ListModesRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
  };
}

export class ListModesResponse extends jspb.Message {
  getModesList(): Array<ElectricMode>;
  setModesList(value: Array<ElectricMode>): ListModesResponse;
  clearModesList(): ListModesResponse;
  addModes(value?: ElectricMode, index?: number): ElectricMode;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListModesResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListModesResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListModesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListModesResponse): ListModesResponse.AsObject;
  static serializeBinaryToWriter(message: ListModesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListModesResponse;
  static deserializeBinaryFromReader(message: ListModesResponse, reader: jspb.BinaryReader): ListModesResponse;
}

export namespace ListModesResponse {
  export type AsObject = {
    modesList: Array<ElectricMode.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class PullModesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullModesRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullModesRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullModesRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullModesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullModesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullModesRequest): PullModesRequest.AsObject;
  static serializeBinaryToWriter(message: PullModesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullModesRequest;
  static deserializeBinaryFromReader(message: PullModesRequest, reader: jspb.BinaryReader): PullModesRequest;
}

export namespace PullModesRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullModesResponse extends jspb.Message {
  getChangesList(): Array<PullModesResponse.Change>;
  setChangesList(value: Array<PullModesResponse.Change>): PullModesResponse;
  clearChangesList(): PullModesResponse;
  addChanges(value?: PullModesResponse.Change, index?: number): PullModesResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullModesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullModesResponse): PullModesResponse.AsObject;
  static serializeBinaryToWriter(message: PullModesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullModesResponse;
  static deserializeBinaryFromReader(message: PullModesResponse, reader: jspb.BinaryReader): PullModesResponse;
}

export namespace PullModesResponse {
  export type AsObject = {
    changesList: Array<PullModesResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getType(): types_change_pb.ChangeType;
    setType(value: types_change_pb.ChangeType): Change;

    getNewValue(): ElectricMode | undefined;
    setNewValue(value?: ElectricMode): Change;
    hasNewValue(): boolean;
    clearNewValue(): Change;

    getOldValue(): ElectricMode | undefined;
    setOldValue(value?: ElectricMode): Change;
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
      newValue?: ElectricMode.AsObject;
      oldValue?: ElectricMode.AsObject;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };
  }

}

