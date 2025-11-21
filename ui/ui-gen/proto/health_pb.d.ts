import * as jspb from 'google-protobuf'

import * as google_protobuf_duration_pb from 'google-protobuf/google/protobuf/duration_pb'; // proto import: "google/protobuf/duration.proto"
import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_change_pb from '@smart-core-os/sc-api-grpc-web/types/change_pb'; // proto import: "types/change.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"


export class HealthCheck extends jspb.Message {
  getId(): string;
  setId(value: string): HealthCheck;

  getDisplayName(): string;
  setDisplayName(value: string): HealthCheck;

  getDescription(): string;
  setDescription(value: string): HealthCheck;

  getCreateTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreateTime(value?: google_protobuf_timestamp_pb.Timestamp): HealthCheck;
  hasCreateTime(): boolean;
  clearCreateTime(): HealthCheck;

  getOccupantImpact(): HealthCheck.OccupantImpact;
  setOccupantImpact(value: HealthCheck.OccupantImpact): HealthCheck;

  getEquipmentImpact(): HealthCheck.EquipmentImpact;
  setEquipmentImpact(value: HealthCheck.EquipmentImpact): HealthCheck;

  getComplianceImpactsList(): Array<HealthCheck.ComplianceImpact>;
  setComplianceImpactsList(value: Array<HealthCheck.ComplianceImpact>): HealthCheck;
  clearComplianceImpactsList(): HealthCheck;
  addComplianceImpacts(value?: HealthCheck.ComplianceImpact, index?: number): HealthCheck.ComplianceImpact;

  getReliability(): HealthCheck.Reliability | undefined;
  setReliability(value?: HealthCheck.Reliability): HealthCheck;
  hasReliability(): boolean;
  clearReliability(): HealthCheck;

  getNormality(): HealthCheck.Normality;
  setNormality(value: HealthCheck.Normality): HealthCheck;

  getNormalTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setNormalTime(value?: google_protobuf_timestamp_pb.Timestamp): HealthCheck;
  hasNormalTime(): boolean;
  clearNormalTime(): HealthCheck;

  getAbnormalTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setAbnormalTime(value?: google_protobuf_timestamp_pb.Timestamp): HealthCheck;
  hasAbnormalTime(): boolean;
  clearAbnormalTime(): HealthCheck;

  getBounds(): HealthCheck.Bounds | undefined;
  setBounds(value?: HealthCheck.Bounds): HealthCheck;
  hasBounds(): boolean;
  clearBounds(): HealthCheck;

  getFaults(): HealthCheck.Faults | undefined;
  setFaults(value?: HealthCheck.Faults): HealthCheck;
  hasFaults(): boolean;
  clearFaults(): HealthCheck;

  getCheckCase(): HealthCheck.CheckCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): HealthCheck.AsObject;
  static toObject(includeInstance: boolean, msg: HealthCheck): HealthCheck.AsObject;
  static serializeBinaryToWriter(message: HealthCheck, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): HealthCheck;
  static deserializeBinaryFromReader(message: HealthCheck, reader: jspb.BinaryReader): HealthCheck;
}

export namespace HealthCheck {
  export type AsObject = {
    id: string,
    displayName: string,
    description: string,
    createTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    occupantImpact: HealthCheck.OccupantImpact,
    equipmentImpact: HealthCheck.EquipmentImpact,
    complianceImpactsList: Array<HealthCheck.ComplianceImpact.AsObject>,
    reliability?: HealthCheck.Reliability.AsObject,
    normality: HealthCheck.Normality,
    normalTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    abnormalTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    bounds?: HealthCheck.Bounds.AsObject,
    faults?: HealthCheck.Faults.AsObject,
  }

  export class ComplianceImpact extends jspb.Message {
    getStandard(): HealthCheck.ComplianceImpact.Standard | undefined;
    setStandard(value?: HealthCheck.ComplianceImpact.Standard): ComplianceImpact;
    hasStandard(): boolean;
    clearStandard(): ComplianceImpact;

    getContribution(): HealthCheck.ComplianceImpact.Contribution;
    setContribution(value: HealthCheck.ComplianceImpact.Contribution): ComplianceImpact;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ComplianceImpact.AsObject;
    static toObject(includeInstance: boolean, msg: ComplianceImpact): ComplianceImpact.AsObject;
    static serializeBinaryToWriter(message: ComplianceImpact, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ComplianceImpact;
    static deserializeBinaryFromReader(message: ComplianceImpact, reader: jspb.BinaryReader): ComplianceImpact;
  }

  export namespace ComplianceImpact {
    export type AsObject = {
      standard?: HealthCheck.ComplianceImpact.Standard.AsObject,
      contribution: HealthCheck.ComplianceImpact.Contribution,
    }

    export class Standard extends jspb.Message {
      getDisplayName(): string;
      setDisplayName(value: string): Standard;

      getTitle(): string;
      setTitle(value: string): Standard;

      getDescription(): string;
      setDescription(value: string): Standard;

      getOrganization(): string;
      setOrganization(value: string): Standard;

      getReference(): string;
      setReference(value: string): Standard;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): Standard.AsObject;
      static toObject(includeInstance: boolean, msg: Standard): Standard.AsObject;
      static serializeBinaryToWriter(message: Standard, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): Standard;
      static deserializeBinaryFromReader(message: Standard, reader: jspb.BinaryReader): Standard;
    }

    export namespace Standard {
      export type AsObject = {
        displayName: string,
        title: string,
        description: string,
        organization: string,
        reference: string,
      }
    }


    export enum Contribution { 
      CONTRIBUTION_UNSPECIFIED = 0,
      NOTE = 1,
      RATING = 2,
      WARNING = 3,
      FAIL = 4,
    }
  }


  export class Error extends jspb.Message {
    getSummaryText(): string;
    setSummaryText(value: string): Error;

    getDetailsText(): string;
    setDetailsText(value: string): Error;

    getCode(): HealthCheck.Error.Code | undefined;
    setCode(value?: HealthCheck.Error.Code): Error;
    hasCode(): boolean;
    clearCode(): Error;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Error.AsObject;
    static toObject(includeInstance: boolean, msg: Error): Error.AsObject;
    static serializeBinaryToWriter(message: Error, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Error;
    static deserializeBinaryFromReader(message: Error, reader: jspb.BinaryReader): Error;
  }

  export namespace Error {
    export type AsObject = {
      summaryText: string,
      detailsText: string,
      code?: HealthCheck.Error.Code.AsObject,
    }

    export class Code extends jspb.Message {
      getCode(): string;
      setCode(value: string): Code;

      getSystem(): string;
      setSystem(value: string): Code;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): Code.AsObject;
      static toObject(includeInstance: boolean, msg: Code): Code.AsObject;
      static serializeBinaryToWriter(message: Code, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): Code;
      static deserializeBinaryFromReader(message: Code, reader: jspb.BinaryReader): Code;
    }

    export namespace Code {
      export type AsObject = {
        code: string,
        system: string,
      }
    }

  }


  export class Reliability extends jspb.Message {
    getState(): HealthCheck.Reliability.State;
    setState(value: HealthCheck.Reliability.State): Reliability;

    getReliableTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setReliableTime(value?: google_protobuf_timestamp_pb.Timestamp): Reliability;
    hasReliableTime(): boolean;
    clearReliableTime(): Reliability;

    getUnreliableTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setUnreliableTime(value?: google_protobuf_timestamp_pb.Timestamp): Reliability;
    hasUnreliableTime(): boolean;
    clearUnreliableTime(): Reliability;

    getLastError(): HealthCheck.Error | undefined;
    setLastError(value?: HealthCheck.Error): Reliability;
    hasLastError(): boolean;
    clearLastError(): Reliability;

    getCause(): HealthCheck.Reliability.Cause | undefined;
    setCause(value?: HealthCheck.Reliability.Cause): Reliability;
    hasCause(): boolean;
    clearCause(): Reliability;

    getEffects(): HealthCheck.Reliability.Effects | undefined;
    setEffects(value?: HealthCheck.Reliability.Effects): Reliability;
    hasEffects(): boolean;
    clearEffects(): Reliability;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Reliability.AsObject;
    static toObject(includeInstance: boolean, msg: Reliability): Reliability.AsObject;
    static serializeBinaryToWriter(message: Reliability, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Reliability;
    static deserializeBinaryFromReader(message: Reliability, reader: jspb.BinaryReader): Reliability;
  }

  export namespace Reliability {
    export type AsObject = {
      state: HealthCheck.Reliability.State,
      reliableTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
      unreliableTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
      lastError?: HealthCheck.Error.AsObject,
      cause?: HealthCheck.Reliability.Cause.AsObject,
      effects?: HealthCheck.Reliability.Effects.AsObject,
    }

    export class Cause extends jspb.Message {
      getName(): string;
      setName(value: string): Cause;

      getError(): HealthCheck.Error | undefined;
      setError(value?: HealthCheck.Error): Cause;
      hasError(): boolean;
      clearError(): Cause;

      getDisplayName(): string;
      setDisplayName(value: string): Cause;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): Cause.AsObject;
      static toObject(includeInstance: boolean, msg: Cause): Cause.AsObject;
      static serializeBinaryToWriter(message: Cause, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): Cause;
      static deserializeBinaryFromReader(message: Cause, reader: jspb.BinaryReader): Cause;
    }

    export namespace Cause {
      export type AsObject = {
        name: string,
        error?: HealthCheck.Error.AsObject,
        displayName: string,
      }
    }


    export class Effects extends jspb.Message {
      getCount(): number;
      setCount(value: number): Effects;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): Effects.AsObject;
      static toObject(includeInstance: boolean, msg: Effects): Effects.AsObject;
      static serializeBinaryToWriter(message: Effects, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): Effects;
      static deserializeBinaryFromReader(message: Effects, reader: jspb.BinaryReader): Effects;
    }

    export namespace Effects {
      export type AsObject = {
        count: number,
      }
    }


    export enum State { 
      STATE_UNSPECIFIED = 0,
      RELIABLE = 1,
      UNRELIABLE = 2,
      CONN_TRANSIENT_FAILURE = 3,
      SEND_FAILURE = 4,
      NO_RESPONSE = 5,
      BAD_RESPONSE = 6,
      NOT_FOUND = 7,
      PERMISSION_DENIED = 8,
    }
  }


  export class Value extends jspb.Message {
    getBoolValue(): boolean;
    setBoolValue(value: boolean): Value;

    getStringValue(): string;
    setStringValue(value: string): Value;

    getIntValue(): number;
    setIntValue(value: number): Value;

    getUintValue(): number;
    setUintValue(value: number): Value;

    getFloatValue(): number;
    setFloatValue(value: number): Value;

    getTimestampValue(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setTimestampValue(value?: google_protobuf_timestamp_pb.Timestamp): Value;
    hasTimestampValue(): boolean;
    clearTimestampValue(): Value;

    getDurationValue(): google_protobuf_duration_pb.Duration | undefined;
    setDurationValue(value?: google_protobuf_duration_pb.Duration): Value;
    hasDurationValue(): boolean;
    clearDurationValue(): Value;

    getValueCase(): Value.ValueCase;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Value.AsObject;
    static toObject(includeInstance: boolean, msg: Value): Value.AsObject;
    static serializeBinaryToWriter(message: Value, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Value;
    static deserializeBinaryFromReader(message: Value, reader: jspb.BinaryReader): Value;
  }

  export namespace Value {
    export type AsObject = {
      boolValue: boolean,
      stringValue: string,
      intValue: number,
      uintValue: number,
      floatValue: number,
      timestampValue?: google_protobuf_timestamp_pb.Timestamp.AsObject,
      durationValue?: google_protobuf_duration_pb.Duration.AsObject,
    }

    export enum ValueCase { 
      VALUE_NOT_SET = 0,
      BOOL_VALUE = 1,
      STRING_VALUE = 2,
      INT_VALUE = 3,
      UINT_VALUE = 4,
      FLOAT_VALUE = 5,
      TIMESTAMP_VALUE = 6,
      DURATION_VALUE = 7,
    }
  }


  export class ValueRange extends jspb.Message {
    getLow(): HealthCheck.Value | undefined;
    setLow(value?: HealthCheck.Value): ValueRange;
    hasLow(): boolean;
    clearLow(): ValueRange;

    getHigh(): HealthCheck.Value | undefined;
    setHigh(value?: HealthCheck.Value): ValueRange;
    hasHigh(): boolean;
    clearHigh(): ValueRange;

    getDeadband(): HealthCheck.Value | undefined;
    setDeadband(value?: HealthCheck.Value): ValueRange;
    hasDeadband(): boolean;
    clearDeadband(): ValueRange;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ValueRange.AsObject;
    static toObject(includeInstance: boolean, msg: ValueRange): ValueRange.AsObject;
    static serializeBinaryToWriter(message: ValueRange, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ValueRange;
    static deserializeBinaryFromReader(message: ValueRange, reader: jspb.BinaryReader): ValueRange;
  }

  export namespace ValueRange {
    export type AsObject = {
      low?: HealthCheck.Value.AsObject,
      high?: HealthCheck.Value.AsObject,
      deadband?: HealthCheck.Value.AsObject,
    }
  }


  export class Values extends jspb.Message {
    getValuesList(): Array<HealthCheck.Value>;
    setValuesList(value: Array<HealthCheck.Value>): Values;
    clearValuesList(): Values;
    addValues(value?: HealthCheck.Value, index?: number): HealthCheck.Value;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Values.AsObject;
    static toObject(includeInstance: boolean, msg: Values): Values.AsObject;
    static serializeBinaryToWriter(message: Values, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Values;
    static deserializeBinaryFromReader(message: Values, reader: jspb.BinaryReader): Values;
  }

  export namespace Values {
    export type AsObject = {
      valuesList: Array<HealthCheck.Value.AsObject>,
    }
  }


  export class Bounds extends jspb.Message {
    getCurrentValue(): HealthCheck.Value | undefined;
    setCurrentValue(value?: HealthCheck.Value): Bounds;
    hasCurrentValue(): boolean;
    clearCurrentValue(): Bounds;

    getNormalValue(): HealthCheck.Value | undefined;
    setNormalValue(value?: HealthCheck.Value): Bounds;
    hasNormalValue(): boolean;
    clearNormalValue(): Bounds;

    getAbnormalValue(): HealthCheck.Value | undefined;
    setAbnormalValue(value?: HealthCheck.Value): Bounds;
    hasAbnormalValue(): boolean;
    clearAbnormalValue(): Bounds;

    getNormalRange(): HealthCheck.ValueRange | undefined;
    setNormalRange(value?: HealthCheck.ValueRange): Bounds;
    hasNormalRange(): boolean;
    clearNormalRange(): Bounds;

    getNormalValues(): HealthCheck.Values | undefined;
    setNormalValues(value?: HealthCheck.Values): Bounds;
    hasNormalValues(): boolean;
    clearNormalValues(): Bounds;

    getAbnormalValues(): HealthCheck.Values | undefined;
    setAbnormalValues(value?: HealthCheck.Values): Bounds;
    hasAbnormalValues(): boolean;
    clearAbnormalValues(): Bounds;

    getDisplayUnit(): string;
    setDisplayUnit(value: string): Bounds;

    getExpectedCase(): Bounds.ExpectedCase;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Bounds.AsObject;
    static toObject(includeInstance: boolean, msg: Bounds): Bounds.AsObject;
    static serializeBinaryToWriter(message: Bounds, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Bounds;
    static deserializeBinaryFromReader(message: Bounds, reader: jspb.BinaryReader): Bounds;
  }

  export namespace Bounds {
    export type AsObject = {
      currentValue?: HealthCheck.Value.AsObject,
      normalValue?: HealthCheck.Value.AsObject,
      abnormalValue?: HealthCheck.Value.AsObject,
      normalRange?: HealthCheck.ValueRange.AsObject,
      normalValues?: HealthCheck.Values.AsObject,
      abnormalValues?: HealthCheck.Values.AsObject,
      displayUnit: string,
    }

    export enum ExpectedCase { 
      EXPECTED_NOT_SET = 0,
      NORMAL_VALUE = 2,
      ABNORMAL_VALUE = 3,
      NORMAL_RANGE = 4,
      NORMAL_VALUES = 6,
      ABNORMAL_VALUES = 7,
    }
  }


  export class Faults extends jspb.Message {
    getCurrentFaultsList(): Array<HealthCheck.Error>;
    setCurrentFaultsList(value: Array<HealthCheck.Error>): Faults;
    clearCurrentFaultsList(): Faults;
    addCurrentFaults(value?: HealthCheck.Error, index?: number): HealthCheck.Error;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Faults.AsObject;
    static toObject(includeInstance: boolean, msg: Faults): Faults.AsObject;
    static serializeBinaryToWriter(message: Faults, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Faults;
    static deserializeBinaryFromReader(message: Faults, reader: jspb.BinaryReader): Faults;
  }

  export namespace Faults {
    export type AsObject = {
      currentFaultsList: Array<HealthCheck.Error.AsObject>,
    }
  }


  export enum OccupantImpact { 
    OCCUPANT_IMPACT_UNSPECIFIED = 0,
    NO_OCCUPANT_IMPACT = 1,
    COMFORT = 2,
    HEALTH = 3,
    LIFE = 4,
  }

  export enum EquipmentImpact { 
    EQUIPMENT_IMPACT_UNSPECIFIED = 0,
    NO_EQUIPMENT_IMPACT = 1,
    WARRANTY = 2,
    LIFESPAN = 3,
    FUNCTION = 4,
  }

  export enum Normality { 
    NORMALITY_UNSPECIFIED = 0,
    NORMAL = 1,
    ABNORMAL = 2,
    LOW = 3,
    HIGH = 4,
  }

  export enum CheckCase { 
    CHECK_NOT_SET = 0,
    BOUNDS = 30,
    FAULTS = 31,
  }
}

export class HealthCheckRecord extends jspb.Message {
  getHealthCheck(): HealthCheck | undefined;
  setHealthCheck(value?: HealthCheck): HealthCheckRecord;
  hasHealthCheck(): boolean;
  clearHealthCheck(): HealthCheckRecord;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): HealthCheckRecord;
  hasRecordTime(): boolean;
  clearRecordTime(): HealthCheckRecord;

  getRecordType(): HealthCheckRecord.RecordType;
  setRecordType(value: HealthCheckRecord.RecordType): HealthCheckRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): HealthCheckRecord.AsObject;
  static toObject(includeInstance: boolean, msg: HealthCheckRecord): HealthCheckRecord.AsObject;
  static serializeBinaryToWriter(message: HealthCheckRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): HealthCheckRecord;
  static deserializeBinaryFromReader(message: HealthCheckRecord, reader: jspb.BinaryReader): HealthCheckRecord;
}

export namespace HealthCheckRecord {
  export type AsObject = {
    healthCheck?: HealthCheck.AsObject,
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    recordType: HealthCheckRecord.RecordType,
  }

  export enum RecordType { 
    RECORD_TYPE_UNSPECIFIED = 0,
    ADDED = 1,
    UPDATED = 2,
    REMOVED = 3,
  }
}

export class ListHealthChecksRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListHealthChecksRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListHealthChecksRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListHealthChecksRequest;

  getPageSize(): number;
  setPageSize(value: number): ListHealthChecksRequest;

  getPageToken(): string;
  setPageToken(value: string): ListHealthChecksRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListHealthChecksRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListHealthChecksRequest): ListHealthChecksRequest.AsObject;
  static serializeBinaryToWriter(message: ListHealthChecksRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListHealthChecksRequest;
  static deserializeBinaryFromReader(message: ListHealthChecksRequest, reader: jspb.BinaryReader): ListHealthChecksRequest;
}

export namespace ListHealthChecksRequest {
  export type AsObject = {
    name: string,
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject,
    pageSize: number,
    pageToken: string,
  }
}

export class ListHealthChecksResponse extends jspb.Message {
  getHealthChecksList(): Array<HealthCheck>;
  setHealthChecksList(value: Array<HealthCheck>): ListHealthChecksResponse;
  clearHealthChecksList(): ListHealthChecksResponse;
  addHealthChecks(value?: HealthCheck, index?: number): HealthCheck;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListHealthChecksResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListHealthChecksResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListHealthChecksResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListHealthChecksResponse): ListHealthChecksResponse.AsObject;
  static serializeBinaryToWriter(message: ListHealthChecksResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListHealthChecksResponse;
  static deserializeBinaryFromReader(message: ListHealthChecksResponse, reader: jspb.BinaryReader): ListHealthChecksResponse;
}

export namespace ListHealthChecksResponse {
  export type AsObject = {
    healthChecksList: Array<HealthCheck.AsObject>,
    nextPageToken: string,
    totalSize: number,
  }
}

export class PullHealthChecksRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullHealthChecksRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullHealthChecksRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullHealthChecksRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullHealthChecksRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullHealthChecksRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullHealthChecksRequest): PullHealthChecksRequest.AsObject;
  static serializeBinaryToWriter(message: PullHealthChecksRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullHealthChecksRequest;
  static deserializeBinaryFromReader(message: PullHealthChecksRequest, reader: jspb.BinaryReader): PullHealthChecksRequest;
}

export namespace PullHealthChecksRequest {
  export type AsObject = {
    name: string,
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject,
    updatesOnly: boolean,
  }
}

export class PullHealthChecksResponse extends jspb.Message {
  getChangesList(): Array<PullHealthChecksResponse.Change>;
  setChangesList(value: Array<PullHealthChecksResponse.Change>): PullHealthChecksResponse;
  clearChangesList(): PullHealthChecksResponse;
  addChanges(value?: PullHealthChecksResponse.Change, index?: number): PullHealthChecksResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullHealthChecksResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullHealthChecksResponse): PullHealthChecksResponse.AsObject;
  static serializeBinaryToWriter(message: PullHealthChecksResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullHealthChecksResponse;
  static deserializeBinaryFromReader(message: PullHealthChecksResponse, reader: jspb.BinaryReader): PullHealthChecksResponse;
}

export namespace PullHealthChecksResponse {
  export type AsObject = {
    changesList: Array<PullHealthChecksResponse.Change.AsObject>,
  }

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getType(): types_change_pb.ChangeType;
    setType(value: types_change_pb.ChangeType): Change;

    getNewValue(): HealthCheck | undefined;
    setNewValue(value?: HealthCheck): Change;
    hasNewValue(): boolean;
    clearNewValue(): Change;

    getOldValue(): HealthCheck | undefined;
    setOldValue(value?: HealthCheck): Change;
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
      name: string,
      type: types_change_pb.ChangeType,
      newValue?: HealthCheck.AsObject,
      oldValue?: HealthCheck.AsObject,
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    }
  }

}

export class GetHealthCheckRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetHealthCheckRequest;

  getId(): string;
  setId(value: string): GetHealthCheckRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetHealthCheckRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetHealthCheckRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetHealthCheckRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetHealthCheckRequest): GetHealthCheckRequest.AsObject;
  static serializeBinaryToWriter(message: GetHealthCheckRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetHealthCheckRequest;
  static deserializeBinaryFromReader(message: GetHealthCheckRequest, reader: jspb.BinaryReader): GetHealthCheckRequest;
}

export namespace GetHealthCheckRequest {
  export type AsObject = {
    name: string,
    id: string,
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject,
  }
}

export class PullHealthCheckRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullHealthCheckRequest;

  getId(): string;
  setId(value: string): PullHealthCheckRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullHealthCheckRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullHealthCheckRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullHealthCheckRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullHealthCheckRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullHealthCheckRequest): PullHealthCheckRequest.AsObject;
  static serializeBinaryToWriter(message: PullHealthCheckRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullHealthCheckRequest;
  static deserializeBinaryFromReader(message: PullHealthCheckRequest, reader: jspb.BinaryReader): PullHealthCheckRequest;
}

export namespace PullHealthCheckRequest {
  export type AsObject = {
    name: string,
    id: string,
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject,
    updatesOnly: boolean,
  }
}

export class PullHealthCheckResponse extends jspb.Message {
  getChangesList(): Array<PullHealthCheckResponse.Change>;
  setChangesList(value: Array<PullHealthCheckResponse.Change>): PullHealthCheckResponse;
  clearChangesList(): PullHealthCheckResponse;
  addChanges(value?: PullHealthCheckResponse.Change, index?: number): PullHealthCheckResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullHealthCheckResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullHealthCheckResponse): PullHealthCheckResponse.AsObject;
  static serializeBinaryToWriter(message: PullHealthCheckResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullHealthCheckResponse;
  static deserializeBinaryFromReader(message: PullHealthCheckResponse, reader: jspb.BinaryReader): PullHealthCheckResponse;
}

export namespace PullHealthCheckResponse {
  export type AsObject = {
    changesList: Array<PullHealthCheckResponse.Change.AsObject>,
  }

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getHealthCheck(): HealthCheck | undefined;
    setHealthCheck(value?: HealthCheck): Change;
    hasHealthCheck(): boolean;
    clearHealthCheck(): Change;

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
      name: string,
      healthCheck?: HealthCheck.AsObject,
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    }
  }

}

export class ListHealthCheckHistoryRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListHealthCheckHistoryRequest;

  getId(): string;
  setId(value: string): ListHealthCheckHistoryRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListHealthCheckHistoryRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListHealthCheckHistoryRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListHealthCheckHistoryRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListHealthCheckHistoryRequest;

  getPageSize(): number;
  setPageSize(value: number): ListHealthCheckHistoryRequest;

  getPageToken(): string;
  setPageToken(value: string): ListHealthCheckHistoryRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListHealthCheckHistoryRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListHealthCheckHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListHealthCheckHistoryRequest): ListHealthCheckHistoryRequest.AsObject;
  static serializeBinaryToWriter(message: ListHealthCheckHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListHealthCheckHistoryRequest;
  static deserializeBinaryFromReader(message: ListHealthCheckHistoryRequest, reader: jspb.BinaryReader): ListHealthCheckHistoryRequest;
}

export namespace ListHealthCheckHistoryRequest {
  export type AsObject = {
    name: string,
    id: string,
    period?: types_time_period_pb.Period.AsObject,
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject,
    pageSize: number,
    pageToken: string,
    orderBy: string,
  }
}

export class ListHealthCheckHistoryResponse extends jspb.Message {
  getHealthCheckRecordsList(): Array<HealthCheckRecord>;
  setHealthCheckRecordsList(value: Array<HealthCheckRecord>): ListHealthCheckHistoryResponse;
  clearHealthCheckRecordsList(): ListHealthCheckHistoryResponse;
  addHealthCheckRecords(value?: HealthCheckRecord, index?: number): HealthCheckRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListHealthCheckHistoryResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListHealthCheckHistoryResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListHealthCheckHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListHealthCheckHistoryResponse): ListHealthCheckHistoryResponse.AsObject;
  static serializeBinaryToWriter(message: ListHealthCheckHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListHealthCheckHistoryResponse;
  static deserializeBinaryFromReader(message: ListHealthCheckHistoryResponse, reader: jspb.BinaryReader): ListHealthCheckHistoryResponse;
}

export namespace ListHealthCheckHistoryResponse {
  export type AsObject = {
    healthCheckRecordsList: Array<HealthCheckRecord.AsObject>,
    nextPageToken: string,
    totalSize: number,
  }
}

