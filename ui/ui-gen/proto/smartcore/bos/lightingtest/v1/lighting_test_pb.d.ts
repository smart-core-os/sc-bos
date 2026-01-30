import * as jspb from 'google-protobuf'

import * as google_protobuf_duration_pb from 'google-protobuf/google/protobuf/duration_pb'; // proto import: "google/protobuf/duration.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class LightHealth extends jspb.Message {
  getName(): string;
  setName(value: string): LightHealth;

  getUpdateTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setUpdateTime(value?: google_protobuf_timestamp_pb.Timestamp): LightHealth;
  hasUpdateTime(): boolean;
  clearUpdateTime(): LightHealth;

  getFaultsList(): Array<LightFault>;
  setFaultsList(value: Array<LightFault>): LightHealth;
  clearFaultsList(): LightHealth;
  addFaults(value: LightFault, index?: number): LightHealth;

  getLastFunctionTest(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastFunctionTest(value?: google_protobuf_timestamp_pb.Timestamp): LightHealth;
  hasLastFunctionTest(): boolean;
  clearLastFunctionTest(): LightHealth;

  getLastDurationTest(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastDurationTest(value?: google_protobuf_timestamp_pb.Timestamp): LightHealth;
  hasLastDurationTest(): boolean;
  clearLastDurationTest(): LightHealth;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LightHealth.AsObject;
  static toObject(includeInstance: boolean, msg: LightHealth): LightHealth.AsObject;
  static serializeBinaryToWriter(message: LightHealth, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LightHealth;
  static deserializeBinaryFromReader(message: LightHealth, reader: jspb.BinaryReader): LightHealth;
}

export namespace LightHealth {
  export type AsObject = {
    name: string;
    updateTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    faultsList: Array<LightFault>;
    lastFunctionTest?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    lastDurationTest?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class LightingEvent extends jspb.Message {
  getName(): string;
  setName(value: string): LightingEvent;

  getId(): string;
  setId(value: string): LightingEvent;

  getTimestamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTimestamp(value?: google_protobuf_timestamp_pb.Timestamp): LightingEvent;
  hasTimestamp(): boolean;
  clearTimestamp(): LightingEvent;

  getDurationTestPass(): LightingEvent.DurationTestPass | undefined;
  setDurationTestPass(value?: LightingEvent.DurationTestPass): LightingEvent;
  hasDurationTestPass(): boolean;
  clearDurationTestPass(): LightingEvent;

  getFunctionTestPass(): LightingEvent.FunctionTestPass | undefined;
  setFunctionTestPass(value?: LightingEvent.FunctionTestPass): LightingEvent;
  hasFunctionTestPass(): boolean;
  clearFunctionTestPass(): LightingEvent;

  getStatusReport(): LightingEvent.StatusReport | undefined;
  setStatusReport(value?: LightingEvent.StatusReport): LightingEvent;
  hasStatusReport(): boolean;
  clearStatusReport(): LightingEvent;

  getEventCase(): LightingEvent.EventCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LightingEvent.AsObject;
  static toObject(includeInstance: boolean, msg: LightingEvent): LightingEvent.AsObject;
  static serializeBinaryToWriter(message: LightingEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LightingEvent;
  static deserializeBinaryFromReader(message: LightingEvent, reader: jspb.BinaryReader): LightingEvent;
}

export namespace LightingEvent {
  export type AsObject = {
    name: string;
    id: string;
    timestamp?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    durationTestPass?: LightingEvent.DurationTestPass.AsObject;
    functionTestPass?: LightingEvent.FunctionTestPass.AsObject;
    statusReport?: LightingEvent.StatusReport.AsObject;
  };

  export class DurationTestPass extends jspb.Message {
    getAchievedDuration(): google_protobuf_duration_pb.Duration | undefined;
    setAchievedDuration(value?: google_protobuf_duration_pb.Duration): DurationTestPass;
    hasAchievedDuration(): boolean;
    clearAchievedDuration(): DurationTestPass;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DurationTestPass.AsObject;
    static toObject(includeInstance: boolean, msg: DurationTestPass): DurationTestPass.AsObject;
    static serializeBinaryToWriter(message: DurationTestPass, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DurationTestPass;
    static deserializeBinaryFromReader(message: DurationTestPass, reader: jspb.BinaryReader): DurationTestPass;
  }

  export namespace DurationTestPass {
    export type AsObject = {
      achievedDuration?: google_protobuf_duration_pb.Duration.AsObject;
    };
  }


  export class FunctionTestPass extends jspb.Message {
    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): FunctionTestPass.AsObject;
    static toObject(includeInstance: boolean, msg: FunctionTestPass): FunctionTestPass.AsObject;
    static serializeBinaryToWriter(message: FunctionTestPass, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): FunctionTestPass;
    static deserializeBinaryFromReader(message: FunctionTestPass, reader: jspb.BinaryReader): FunctionTestPass;
  }

  export namespace FunctionTestPass {
    export type AsObject = {
    };
  }


  export class StatusReport extends jspb.Message {
    getFaultsList(): Array<LightFault>;
    setFaultsList(value: Array<LightFault>): StatusReport;
    clearFaultsList(): StatusReport;
    addFaults(value: LightFault, index?: number): StatusReport;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): StatusReport.AsObject;
    static toObject(includeInstance: boolean, msg: StatusReport): StatusReport.AsObject;
    static serializeBinaryToWriter(message: StatusReport, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): StatusReport;
    static deserializeBinaryFromReader(message: StatusReport, reader: jspb.BinaryReader): StatusReport;
  }

  export namespace StatusReport {
    export type AsObject = {
      faultsList: Array<LightFault>;
    };
  }


  export enum EventCase {
    EVENT_NOT_SET = 0,
    DURATION_TEST_PASS = 4,
    FUNCTION_TEST_PASS = 5,
    STATUS_REPORT = 6,
  }
}

export class GetLightHealthRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetLightHealthRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetLightHealthRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetLightHealthRequest): GetLightHealthRequest.AsObject;
  static serializeBinaryToWriter(message: GetLightHealthRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetLightHealthRequest;
  static deserializeBinaryFromReader(message: GetLightHealthRequest, reader: jspb.BinaryReader): GetLightHealthRequest;
}

export namespace GetLightHealthRequest {
  export type AsObject = {
    name: string;
  };
}

export class ListLightHealthRequest extends jspb.Message {
  getPageSize(): number;
  setPageSize(value: number): ListLightHealthRequest;

  getPageToken(): string;
  setPageToken(value: string): ListLightHealthRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListLightHealthRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListLightHealthRequest): ListLightHealthRequest.AsObject;
  static serializeBinaryToWriter(message: ListLightHealthRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListLightHealthRequest;
  static deserializeBinaryFromReader(message: ListLightHealthRequest, reader: jspb.BinaryReader): ListLightHealthRequest;
}

export namespace ListLightHealthRequest {
  export type AsObject = {
    pageSize: number;
    pageToken: string;
  };
}

export class ListLightHealthResponse extends jspb.Message {
  getEmergencyLightsList(): Array<LightHealth>;
  setEmergencyLightsList(value: Array<LightHealth>): ListLightHealthResponse;
  clearEmergencyLightsList(): ListLightHealthResponse;
  addEmergencyLights(value?: LightHealth, index?: number): LightHealth;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListLightHealthResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListLightHealthResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListLightHealthResponse): ListLightHealthResponse.AsObject;
  static serializeBinaryToWriter(message: ListLightHealthResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListLightHealthResponse;
  static deserializeBinaryFromReader(message: ListLightHealthResponse, reader: jspb.BinaryReader): ListLightHealthResponse;
}

export namespace ListLightHealthResponse {
  export type AsObject = {
    emergencyLightsList: Array<LightHealth.AsObject>;
    nextPageToken: string;
  };
}

export class ListLightEventsRequest extends jspb.Message {
  getPageSize(): number;
  setPageSize(value: number): ListLightEventsRequest;

  getPageToken(): string;
  setPageToken(value: string): ListLightEventsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListLightEventsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListLightEventsRequest): ListLightEventsRequest.AsObject;
  static serializeBinaryToWriter(message: ListLightEventsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListLightEventsRequest;
  static deserializeBinaryFromReader(message: ListLightEventsRequest, reader: jspb.BinaryReader): ListLightEventsRequest;
}

export namespace ListLightEventsRequest {
  export type AsObject = {
    pageSize: number;
    pageToken: string;
  };
}

export class ListLightEventsResponse extends jspb.Message {
  getEventsList(): Array<LightingEvent>;
  setEventsList(value: Array<LightingEvent>): ListLightEventsResponse;
  clearEventsList(): ListLightEventsResponse;
  addEvents(value?: LightingEvent, index?: number): LightingEvent;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListLightEventsResponse;

  getFuturePageToken(): string;
  setFuturePageToken(value: string): ListLightEventsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListLightEventsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListLightEventsResponse): ListLightEventsResponse.AsObject;
  static serializeBinaryToWriter(message: ListLightEventsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListLightEventsResponse;
  static deserializeBinaryFromReader(message: ListLightEventsResponse, reader: jspb.BinaryReader): ListLightEventsResponse;
}

export namespace ListLightEventsResponse {
  export type AsObject = {
    eventsList: Array<LightingEvent.AsObject>;
    nextPageToken: string;
    futurePageToken: string;
  };
}

export class GetReportCSVRequest extends jspb.Message {
  getIncludeHeader(): boolean;
  setIncludeHeader(value: boolean): GetReportCSVRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetReportCSVRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetReportCSVRequest): GetReportCSVRequest.AsObject;
  static serializeBinaryToWriter(message: GetReportCSVRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetReportCSVRequest;
  static deserializeBinaryFromReader(message: GetReportCSVRequest, reader: jspb.BinaryReader): GetReportCSVRequest;
}

export namespace GetReportCSVRequest {
  export type AsObject = {
    includeHeader: boolean;
  };
}

export class ReportCSV extends jspb.Message {
  getCsv(): Uint8Array | string;
  getCsv_asU8(): Uint8Array;
  getCsv_asB64(): string;
  setCsv(value: Uint8Array | string): ReportCSV;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ReportCSV.AsObject;
  static toObject(includeInstance: boolean, msg: ReportCSV): ReportCSV.AsObject;
  static serializeBinaryToWriter(message: ReportCSV, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ReportCSV;
  static deserializeBinaryFromReader(message: ReportCSV, reader: jspb.BinaryReader): ReportCSV;
}

export namespace ReportCSV {
  export type AsObject = {
    csv: Uint8Array | string;
  };
}

export enum LightFault {
  FAULT_UNSPECIFIED = 0,
  DURATION_TEST_FAILED = 1,
  FUNCTION_TEST_FAILED = 2,
  BATTERY_FAULT = 3,
  LAMP_FAULT = 4,
  COMMUNICATION_FAILURE = 5,
  OTHER_FAULT = 6,
}
