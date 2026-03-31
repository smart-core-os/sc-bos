import * as jspb from 'google-protobuf'

import * as google_protobuf_duration_pb from 'google-protobuf/google/protobuf/duration_pb'; // proto import: "google/protobuf/duration.proto"
import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class TestResultSet extends jspb.Message {
  getFunctionTest(): EmergencyTestResult | undefined;
  setFunctionTest(value?: EmergencyTestResult): TestResultSet;
  hasFunctionTest(): boolean;
  clearFunctionTest(): TestResultSet;

  getDurationTest(): EmergencyTestResult | undefined;
  setDurationTest(value?: EmergencyTestResult): TestResultSet;
  hasDurationTest(): boolean;
  clearDurationTest(): TestResultSet;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TestResultSet.AsObject;
  static toObject(includeInstance: boolean, msg: TestResultSet): TestResultSet.AsObject;
  static serializeBinaryToWriter(message: TestResultSet, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TestResultSet;
  static deserializeBinaryFromReader(message: TestResultSet, reader: jspb.BinaryReader): TestResultSet;
}

export namespace TestResultSet {
  export type AsObject = {
    functionTest?: EmergencyTestResult.AsObject;
    durationTest?: EmergencyTestResult.AsObject;
  };
}

export class EmergencyTestResult extends jspb.Message {
  getResult(): EmergencyTestResult.Result;
  setResult(value: EmergencyTestResult.Result): EmergencyTestResult;

  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): EmergencyTestResult;
  hasStartTime(): boolean;
  clearStartTime(): EmergencyTestResult;

  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): EmergencyTestResult;
  hasEndTime(): boolean;
  clearEndTime(): EmergencyTestResult;

  getDuration(): google_protobuf_duration_pb.Duration | undefined;
  setDuration(value?: google_protobuf_duration_pb.Duration): EmergencyTestResult;
  hasDuration(): boolean;
  clearDuration(): EmergencyTestResult;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): EmergencyTestResult.AsObject;
  static toObject(includeInstance: boolean, msg: EmergencyTestResult): EmergencyTestResult.AsObject;
  static serializeBinaryToWriter(message: EmergencyTestResult, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): EmergencyTestResult;
  static deserializeBinaryFromReader(message: EmergencyTestResult, reader: jspb.BinaryReader): EmergencyTestResult;
}

export namespace EmergencyTestResult {
  export type AsObject = {
    result: EmergencyTestResult.Result;
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    duration?: google_protobuf_duration_pb.Duration.AsObject;
  };

  export enum Result {
    TEST_RESULT_UNSPECIFIED = 0,
    TEST_RESULT_PENDING = 1,
    TEST_PASSED = 2,
    CIRCUIT_FAILURE = 3,
    BATTERY_DURATION_FAILURE = 4,
    BATTERY_FAILURE = 5,
    LAMP_FAILURE = 6,
    TEST_FAILED = 7,
    LIGHT_FAULTY = 8,
    COMMUNICATION_FAILURE = 9,
    OTHER_FAULT = 10,
  }
}

export class StartEmergencyTestRequest extends jspb.Message {
  getName(): string;
  setName(value: string): StartEmergencyTestRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StartEmergencyTestRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StartEmergencyTestRequest): StartEmergencyTestRequest.AsObject;
  static serializeBinaryToWriter(message: StartEmergencyTestRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StartEmergencyTestRequest;
  static deserializeBinaryFromReader(message: StartEmergencyTestRequest, reader: jspb.BinaryReader): StartEmergencyTestRequest;
}

export namespace StartEmergencyTestRequest {
  export type AsObject = {
    name: string;
  };
}

export class StartEmergencyTestResponse extends jspb.Message {
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): StartEmergencyTestResponse;
  hasStartTime(): boolean;
  clearStartTime(): StartEmergencyTestResponse;

  getDuration(): google_protobuf_duration_pb.Duration | undefined;
  setDuration(value?: google_protobuf_duration_pb.Duration): StartEmergencyTestResponse;
  hasDuration(): boolean;
  clearDuration(): StartEmergencyTestResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StartEmergencyTestResponse.AsObject;
  static toObject(includeInstance: boolean, msg: StartEmergencyTestResponse): StartEmergencyTestResponse.AsObject;
  static serializeBinaryToWriter(message: StartEmergencyTestResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StartEmergencyTestResponse;
  static deserializeBinaryFromReader(message: StartEmergencyTestResponse, reader: jspb.BinaryReader): StartEmergencyTestResponse;
}

export namespace StartEmergencyTestResponse {
  export type AsObject = {
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    duration?: google_protobuf_duration_pb.Duration.AsObject;
  };
}

export class StopEmergencyTestsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): StopEmergencyTestsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StopEmergencyTestsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StopEmergencyTestsRequest): StopEmergencyTestsRequest.AsObject;
  static serializeBinaryToWriter(message: StopEmergencyTestsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StopEmergencyTestsRequest;
  static deserializeBinaryFromReader(message: StopEmergencyTestsRequest, reader: jspb.BinaryReader): StopEmergencyTestsRequest;
}

export namespace StopEmergencyTestsRequest {
  export type AsObject = {
    name: string;
  };
}

export class StopEmergencyTestsResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StopEmergencyTestsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: StopEmergencyTestsResponse): StopEmergencyTestsResponse.AsObject;
  static serializeBinaryToWriter(message: StopEmergencyTestsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StopEmergencyTestsResponse;
  static deserializeBinaryFromReader(message: StopEmergencyTestsResponse, reader: jspb.BinaryReader): StopEmergencyTestsResponse;
}

export namespace StopEmergencyTestsResponse {
  export type AsObject = {
  };
}

export class GetTestResultSetRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetTestResultSetRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetTestResultSetRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetTestResultSetRequest;

  getQueryDevice(): boolean;
  setQueryDevice(value: boolean): GetTestResultSetRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetTestResultSetRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetTestResultSetRequest): GetTestResultSetRequest.AsObject;
  static serializeBinaryToWriter(message: GetTestResultSetRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetTestResultSetRequest;
  static deserializeBinaryFromReader(message: GetTestResultSetRequest, reader: jspb.BinaryReader): GetTestResultSetRequest;
}

export namespace GetTestResultSetRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    queryDevice: boolean;
  };
}

export class PullTestResultRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullTestResultRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullTestResultRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullTestResultRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullTestResultRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullTestResultRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullTestResultRequest): PullTestResultRequest.AsObject;
  static serializeBinaryToWriter(message: PullTestResultRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullTestResultRequest;
  static deserializeBinaryFromReader(message: PullTestResultRequest, reader: jspb.BinaryReader): PullTestResultRequest;
}

export namespace PullTestResultRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullTestResultsResponse extends jspb.Message {
  getChangesList(): Array<PullTestResultsResponse.Change>;
  setChangesList(value: Array<PullTestResultsResponse.Change>): PullTestResultsResponse;
  clearChangesList(): PullTestResultsResponse;
  addChanges(value?: PullTestResultsResponse.Change, index?: number): PullTestResultsResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullTestResultsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullTestResultsResponse): PullTestResultsResponse.AsObject;
  static serializeBinaryToWriter(message: PullTestResultsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullTestResultsResponse;
  static deserializeBinaryFromReader(message: PullTestResultsResponse, reader: jspb.BinaryReader): PullTestResultsResponse;
}

export namespace PullTestResultsResponse {
  export type AsObject = {
    changesList: Array<PullTestResultsResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getTestResult(): TestResultSet | undefined;
    setTestResult(value?: TestResultSet): Change;
    hasTestResult(): boolean;
    clearTestResult(): Change;

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
      testResult?: TestResultSet.AsObject;
    };
  }

}

