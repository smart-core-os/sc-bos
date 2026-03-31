import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_types_v1_info_pb from '../../../../smartcore/bos/types/v1/info_pb'; // proto import: "smartcore/bos/types/v1/info.proto"
import * as smartcore_bos_types_v1_number_pb from '../../../../smartcore/bos/types/v1/number_pb'; // proto import: "smartcore/bos/types/v1/number.proto"


export class AirQuality extends jspb.Message {
  getCarbonDioxideLevel(): number;
  setCarbonDioxideLevel(value: number): AirQuality;
  hasCarbonDioxideLevel(): boolean;
  clearCarbonDioxideLevel(): AirQuality;

  getVolatileOrganicCompounds(): number;
  setVolatileOrganicCompounds(value: number): AirQuality;
  hasVolatileOrganicCompounds(): boolean;
  clearVolatileOrganicCompounds(): AirQuality;

  getAirPressure(): number;
  setAirPressure(value: number): AirQuality;
  hasAirPressure(): boolean;
  clearAirPressure(): AirQuality;

  getComfort(): AirQuality.Comfort;
  setComfort(value: AirQuality.Comfort): AirQuality;

  getInfectionRisk(): number;
  setInfectionRisk(value: number): AirQuality;
  hasInfectionRisk(): boolean;
  clearInfectionRisk(): AirQuality;

  getScore(): number;
  setScore(value: number): AirQuality;
  hasScore(): boolean;
  clearScore(): AirQuality;

  getParticulateMatter1(): number;
  setParticulateMatter1(value: number): AirQuality;
  hasParticulateMatter1(): boolean;
  clearParticulateMatter1(): AirQuality;

  getParticulateMatter25(): number;
  setParticulateMatter25(value: number): AirQuality;
  hasParticulateMatter25(): boolean;
  clearParticulateMatter25(): AirQuality;

  getParticulateMatter10(): number;
  setParticulateMatter10(value: number): AirQuality;
  hasParticulateMatter10(): boolean;
  clearParticulateMatter10(): AirQuality;

  getAirChangePerHour(): number;
  setAirChangePerHour(value: number): AirQuality;
  hasAirChangePerHour(): boolean;
  clearAirChangePerHour(): AirQuality;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AirQuality.AsObject;
  static toObject(includeInstance: boolean, msg: AirQuality): AirQuality.AsObject;
  static serializeBinaryToWriter(message: AirQuality, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AirQuality;
  static deserializeBinaryFromReader(message: AirQuality, reader: jspb.BinaryReader): AirQuality;
}

export namespace AirQuality {
  export type AsObject = {
    carbonDioxideLevel?: number;
    volatileOrganicCompounds?: number;
    airPressure?: number;
    comfort: AirQuality.Comfort;
    infectionRisk?: number;
    score?: number;
    particulateMatter1?: number;
    particulateMatter25?: number;
    particulateMatter10?: number;
    airChangePerHour?: number;
  };

  export enum Comfort {
    COMFORT_UNSPECIFIED = 0,
    COMFORTABLE = 1,
    UNCOMFORTABLE = 2,
  }

  export enum CarbonDioxideLevelCase {
    _CARBON_DIOXIDE_LEVEL_NOT_SET = 0,
    CARBON_DIOXIDE_LEVEL = 1,
  }

  export enum VolatileOrganicCompoundsCase {
    _VOLATILE_ORGANIC_COMPOUNDS_NOT_SET = 0,
    VOLATILE_ORGANIC_COMPOUNDS = 2,
  }

  export enum AirPressureCase {
    _AIR_PRESSURE_NOT_SET = 0,
    AIR_PRESSURE = 3,
  }

  export enum InfectionRiskCase {
    _INFECTION_RISK_NOT_SET = 0,
    INFECTION_RISK = 5,
  }

  export enum ScoreCase {
    _SCORE_NOT_SET = 0,
    SCORE = 6,
  }

  export enum ParticulateMatter1Case {
    _PARTICULATE_MATTER_1_NOT_SET = 0,
    PARTICULATE_MATTER_1 = 7,
  }

  export enum ParticulateMatter25Case {
    _PARTICULATE_MATTER_25_NOT_SET = 0,
    PARTICULATE_MATTER_25 = 8,
  }

  export enum ParticulateMatter10Case {
    _PARTICULATE_MATTER_10_NOT_SET = 0,
    PARTICULATE_MATTER_10 = 9,
  }

  export enum AirChangePerHourCase {
    _AIR_CHANGE_PER_HOUR_NOT_SET = 0,
    AIR_CHANGE_PER_HOUR = 10,
  }
}

export class AirQualitySupport extends jspb.Message {
  getResourceSupport(): smartcore_bos_types_v1_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: smartcore_bos_types_v1_info_pb.ResourceSupport): AirQualitySupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): AirQualitySupport;

  getCarbonDioxideLevel(): smartcore_bos_types_v1_number_pb.FloatBounds | undefined;
  setCarbonDioxideLevel(value?: smartcore_bos_types_v1_number_pb.FloatBounds): AirQualitySupport;
  hasCarbonDioxideLevel(): boolean;
  clearCarbonDioxideLevel(): AirQualitySupport;

  getVolatileOrganicCompounds(): smartcore_bos_types_v1_number_pb.FloatBounds | undefined;
  setVolatileOrganicCompounds(value?: smartcore_bos_types_v1_number_pb.FloatBounds): AirQualitySupport;
  hasVolatileOrganicCompounds(): boolean;
  clearVolatileOrganicCompounds(): AirQualitySupport;

  getAirPressure(): smartcore_bos_types_v1_number_pb.FloatBounds | undefined;
  setAirPressure(value?: smartcore_bos_types_v1_number_pb.FloatBounds): AirQualitySupport;
  hasAirPressure(): boolean;
  clearAirPressure(): AirQualitySupport;

  getComfortList(): Array<AirQuality.Comfort>;
  setComfortList(value: Array<AirQuality.Comfort>): AirQualitySupport;
  clearComfortList(): AirQualitySupport;
  addComfort(value: AirQuality.Comfort, index?: number): AirQualitySupport;

  getInfectionRisk(): smartcore_bos_types_v1_number_pb.FloatBounds | undefined;
  setInfectionRisk(value?: smartcore_bos_types_v1_number_pb.FloatBounds): AirQualitySupport;
  hasInfectionRisk(): boolean;
  clearInfectionRisk(): AirQualitySupport;

  getScore(): smartcore_bos_types_v1_number_pb.FloatBounds | undefined;
  setScore(value?: smartcore_bos_types_v1_number_pb.FloatBounds): AirQualitySupport;
  hasScore(): boolean;
  clearScore(): AirQualitySupport;

  getParticulateMatter1(): smartcore_bos_types_v1_number_pb.FloatBounds | undefined;
  setParticulateMatter1(value?: smartcore_bos_types_v1_number_pb.FloatBounds): AirQualitySupport;
  hasParticulateMatter1(): boolean;
  clearParticulateMatter1(): AirQualitySupport;

  getParticulateMatter25(): smartcore_bos_types_v1_number_pb.FloatBounds | undefined;
  setParticulateMatter25(value?: smartcore_bos_types_v1_number_pb.FloatBounds): AirQualitySupport;
  hasParticulateMatter25(): boolean;
  clearParticulateMatter25(): AirQualitySupport;

  getParticulateMatter10(): smartcore_bos_types_v1_number_pb.FloatBounds | undefined;
  setParticulateMatter10(value?: smartcore_bos_types_v1_number_pb.FloatBounds): AirQualitySupport;
  hasParticulateMatter10(): boolean;
  clearParticulateMatter10(): AirQualitySupport;

  getAirChangePerHour(): smartcore_bos_types_v1_number_pb.FloatBounds | undefined;
  setAirChangePerHour(value?: smartcore_bos_types_v1_number_pb.FloatBounds): AirQualitySupport;
  hasAirChangePerHour(): boolean;
  clearAirChangePerHour(): AirQualitySupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AirQualitySupport.AsObject;
  static toObject(includeInstance: boolean, msg: AirQualitySupport): AirQualitySupport.AsObject;
  static serializeBinaryToWriter(message: AirQualitySupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AirQualitySupport;
  static deserializeBinaryFromReader(message: AirQualitySupport, reader: jspb.BinaryReader): AirQualitySupport;
}

export namespace AirQualitySupport {
  export type AsObject = {
    resourceSupport?: smartcore_bos_types_v1_info_pb.ResourceSupport.AsObject;
    carbonDioxideLevel?: smartcore_bos_types_v1_number_pb.FloatBounds.AsObject;
    volatileOrganicCompounds?: smartcore_bos_types_v1_number_pb.FloatBounds.AsObject;
    airPressure?: smartcore_bos_types_v1_number_pb.FloatBounds.AsObject;
    comfortList: Array<AirQuality.Comfort>;
    infectionRisk?: smartcore_bos_types_v1_number_pb.FloatBounds.AsObject;
    score?: smartcore_bos_types_v1_number_pb.FloatBounds.AsObject;
    particulateMatter1?: smartcore_bos_types_v1_number_pb.FloatBounds.AsObject;
    particulateMatter25?: smartcore_bos_types_v1_number_pb.FloatBounds.AsObject;
    particulateMatter10?: smartcore_bos_types_v1_number_pb.FloatBounds.AsObject;
    airChangePerHour?: smartcore_bos_types_v1_number_pb.FloatBounds.AsObject;
  };
}

export class GetAirQualityRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetAirQualityRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetAirQualityRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetAirQualityRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetAirQualityRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetAirQualityRequest): GetAirQualityRequest.AsObject;
  static serializeBinaryToWriter(message: GetAirQualityRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetAirQualityRequest;
  static deserializeBinaryFromReader(message: GetAirQualityRequest, reader: jspb.BinaryReader): GetAirQualityRequest;
}

export namespace GetAirQualityRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullAirQualityRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullAirQualityRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullAirQualityRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullAirQualityRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullAirQualityRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullAirQualityRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullAirQualityRequest): PullAirQualityRequest.AsObject;
  static serializeBinaryToWriter(message: PullAirQualityRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullAirQualityRequest;
  static deserializeBinaryFromReader(message: PullAirQualityRequest, reader: jspb.BinaryReader): PullAirQualityRequest;
}

export namespace PullAirQualityRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullAirQualityResponse extends jspb.Message {
  getChangesList(): Array<PullAirQualityResponse.Change>;
  setChangesList(value: Array<PullAirQualityResponse.Change>): PullAirQualityResponse;
  clearChangesList(): PullAirQualityResponse;
  addChanges(value?: PullAirQualityResponse.Change, index?: number): PullAirQualityResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullAirQualityResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullAirQualityResponse): PullAirQualityResponse.AsObject;
  static serializeBinaryToWriter(message: PullAirQualityResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullAirQualityResponse;
  static deserializeBinaryFromReader(message: PullAirQualityResponse, reader: jspb.BinaryReader): PullAirQualityResponse;
}

export namespace PullAirQualityResponse {
  export type AsObject = {
    changesList: Array<PullAirQualityResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getAirQuality(): AirQuality | undefined;
    setAirQuality(value?: AirQuality): Change;
    hasAirQuality(): boolean;
    clearAirQuality(): Change;

    getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
    setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): Change;
    hasUpdateMask(): boolean;
    clearUpdateMask(): Change;

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
      airQuality?: AirQuality.AsObject;
      updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    };
  }

}

export class DescribeAirQualityRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeAirQualityRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeAirQualityRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeAirQualityRequest): DescribeAirQualityRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeAirQualityRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeAirQualityRequest;
  static deserializeBinaryFromReader(message: DescribeAirQualityRequest, reader: jspb.BinaryReader): DescribeAirQualityRequest;
}

export namespace DescribeAirQualityRequest {
  export type AsObject = {
    name: string;
  };
}

