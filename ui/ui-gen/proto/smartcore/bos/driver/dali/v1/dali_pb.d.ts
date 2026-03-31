import * as jspb from 'google-protobuf'

import * as google_protobuf_duration_pb from 'google-protobuf/google/protobuf/duration_pb'; // proto import: "google/protobuf/duration.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class GetGroupMembershipRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetGroupMembershipRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetGroupMembershipRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetGroupMembershipRequest): GetGroupMembershipRequest.AsObject;
  static serializeBinaryToWriter(message: GetGroupMembershipRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetGroupMembershipRequest;
  static deserializeBinaryFromReader(message: GetGroupMembershipRequest, reader: jspb.BinaryReader): GetGroupMembershipRequest;
}

export namespace GetGroupMembershipRequest {
  export type AsObject = {
    name: string;
  };
}

export class GetGroupMembershipResponse extends jspb.Message {
  getGroupsList(): Array<number>;
  setGroupsList(value: Array<number>): GetGroupMembershipResponse;
  clearGroupsList(): GetGroupMembershipResponse;
  addGroups(value: number, index?: number): GetGroupMembershipResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetGroupMembershipResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetGroupMembershipResponse): GetGroupMembershipResponse.AsObject;
  static serializeBinaryToWriter(message: GetGroupMembershipResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetGroupMembershipResponse;
  static deserializeBinaryFromReader(message: GetGroupMembershipResponse, reader: jspb.BinaryReader): GetGroupMembershipResponse;
}

export namespace GetGroupMembershipResponse {
  export type AsObject = {
    groupsList: Array<number>;
  };
}

export class AddToGroupRequest extends jspb.Message {
  getName(): string;
  setName(value: string): AddToGroupRequest;

  getGroup(): number;
  setGroup(value: number): AddToGroupRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AddToGroupRequest.AsObject;
  static toObject(includeInstance: boolean, msg: AddToGroupRequest): AddToGroupRequest.AsObject;
  static serializeBinaryToWriter(message: AddToGroupRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AddToGroupRequest;
  static deserializeBinaryFromReader(message: AddToGroupRequest, reader: jspb.BinaryReader): AddToGroupRequest;
}

export namespace AddToGroupRequest {
  export type AsObject = {
    name: string;
    group: number;
  };
}

export class AddToGroupResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AddToGroupResponse.AsObject;
  static toObject(includeInstance: boolean, msg: AddToGroupResponse): AddToGroupResponse.AsObject;
  static serializeBinaryToWriter(message: AddToGroupResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AddToGroupResponse;
  static deserializeBinaryFromReader(message: AddToGroupResponse, reader: jspb.BinaryReader): AddToGroupResponse;
}

export namespace AddToGroupResponse {
  export type AsObject = {
  };
}

export class RemoveFromGroupRequest extends jspb.Message {
  getName(): string;
  setName(value: string): RemoveFromGroupRequest;

  getGroup(): number;
  setGroup(value: number): RemoveFromGroupRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RemoveFromGroupRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RemoveFromGroupRequest): RemoveFromGroupRequest.AsObject;
  static serializeBinaryToWriter(message: RemoveFromGroupRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RemoveFromGroupRequest;
  static deserializeBinaryFromReader(message: RemoveFromGroupRequest, reader: jspb.BinaryReader): RemoveFromGroupRequest;
}

export namespace RemoveFromGroupRequest {
  export type AsObject = {
    name: string;
    group: number;
  };
}

export class RemoveFromGroupResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RemoveFromGroupResponse.AsObject;
  static toObject(includeInstance: boolean, msg: RemoveFromGroupResponse): RemoveFromGroupResponse.AsObject;
  static serializeBinaryToWriter(message: RemoveFromGroupResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RemoveFromGroupResponse;
  static deserializeBinaryFromReader(message: RemoveFromGroupResponse, reader: jspb.BinaryReader): RemoveFromGroupResponse;
}

export namespace RemoveFromGroupResponse {
  export type AsObject = {
  };
}

export class EmergencyStatus extends jspb.Message {
  getActiveModesList(): Array<EmergencyStatus.Mode>;
  setActiveModesList(value: Array<EmergencyStatus.Mode>): EmergencyStatus;
  clearActiveModesList(): EmergencyStatus;
  addActiveModes(value: EmergencyStatus.Mode, index?: number): EmergencyStatus;

  getPendingTestsList(): Array<EmergencyStatus.Test>;
  setPendingTestsList(value: Array<EmergencyStatus.Test>): EmergencyStatus;
  clearPendingTestsList(): EmergencyStatus;
  addPendingTests(value: EmergencyStatus.Test, index?: number): EmergencyStatus;

  getOverdueTestsList(): Array<EmergencyStatus.Test>;
  setOverdueTestsList(value: Array<EmergencyStatus.Test>): EmergencyStatus;
  clearOverdueTestsList(): EmergencyStatus;
  addOverdueTests(value: EmergencyStatus.Test, index?: number): EmergencyStatus;

  getResultsAvailableList(): Array<EmergencyStatus.Test>;
  setResultsAvailableList(value: Array<EmergencyStatus.Test>): EmergencyStatus;
  clearResultsAvailableList(): EmergencyStatus;
  addResultsAvailable(value: EmergencyStatus.Test, index?: number): EmergencyStatus;

  getInhibitActive(): boolean;
  setInhibitActive(value: boolean): EmergencyStatus;

  getIdentificationActive(): boolean;
  setIdentificationActive(value: boolean): EmergencyStatus;

  getBatteryLevelPercent(): number;
  setBatteryLevelPercent(value: number): EmergencyStatus;

  getFailuresList(): Array<EmergencyStatus.Failure>;
  setFailuresList(value: Array<EmergencyStatus.Failure>): EmergencyStatus;
  clearFailuresList(): EmergencyStatus;
  addFailures(value: EmergencyStatus.Failure, index?: number): EmergencyStatus;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): EmergencyStatus.AsObject;
  static toObject(includeInstance: boolean, msg: EmergencyStatus): EmergencyStatus.AsObject;
  static serializeBinaryToWriter(message: EmergencyStatus, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): EmergencyStatus;
  static deserializeBinaryFromReader(message: EmergencyStatus, reader: jspb.BinaryReader): EmergencyStatus;
}

export namespace EmergencyStatus {
  export type AsObject = {
    activeModesList: Array<EmergencyStatus.Mode>;
    pendingTestsList: Array<EmergencyStatus.Test>;
    overdueTestsList: Array<EmergencyStatus.Test>;
    resultsAvailableList: Array<EmergencyStatus.Test>;
    inhibitActive: boolean;
    identificationActive: boolean;
    batteryLevelPercent: number;
    failuresList: Array<EmergencyStatus.Failure>;
  };

  export enum Test {
    TEST_UNKNOWN = 0,
    NO_TEST = 1,
    FUNCTION_TEST = 2,
    DURATION_TEST = 3,
  }

  export enum Mode {
    MODE_UNSPECIFIED = 0,
    REST = 1,
    NORMAL = 3,
    EMERGENCY = 4,
    EXTENDED_EMERGENCY = 5,
    FUNCTION_TEST_ACTIVE = 6,
    DURATION_TEST_ACTIVE = 7,
    HARDWIRED_INHIBIT = 8,
    HARDWIRED_SWITCH = 9,
  }

  export enum Failure {
    FAILURE_UNSPECIFIED = 0,
    CIRCUIT_FAILURE = 1,
    BATTERY_DURATION_FAILURE = 2,
    BATTERY_FAILURE = 3,
    LAMP_FAILURE = 4,
    FUNCTION_TEST_FAILED = 5,
    DURATION_TEST_FAILED = 6,
  }
}

export class GetEmergencyStatusRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetEmergencyStatusRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEmergencyStatusRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetEmergencyStatusRequest): GetEmergencyStatusRequest.AsObject;
  static serializeBinaryToWriter(message: GetEmergencyStatusRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetEmergencyStatusRequest;
  static deserializeBinaryFromReader(message: GetEmergencyStatusRequest, reader: jspb.BinaryReader): GetEmergencyStatusRequest;
}

export namespace GetEmergencyStatusRequest {
  export type AsObject = {
    name: string;
  };
}

export class ControlGearStatus extends jspb.Message {
  getFailuresList(): Array<ControlGearStatus.Failure>;
  setFailuresList(value: Array<ControlGearStatus.Failure>): ControlGearStatus;
  clearFailuresList(): ControlGearStatus;
  addFailures(value: ControlGearStatus.Failure, index?: number): ControlGearStatus;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ControlGearStatus.AsObject;
  static toObject(includeInstance: boolean, msg: ControlGearStatus): ControlGearStatus.AsObject;
  static serializeBinaryToWriter(message: ControlGearStatus, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ControlGearStatus;
  static deserializeBinaryFromReader(message: ControlGearStatus, reader: jspb.BinaryReader): ControlGearStatus;
}

export namespace ControlGearStatus {
  export type AsObject = {
    failuresList: Array<ControlGearStatus.Failure>;
  };

  export enum Failure {
    FAILURE_UNSPECIFIED = 0,
    LAMP_FAILURE = 1,
    CONTROL_GEAR_FAILURE = 2,
  }
}

export class GetControlGearStatusRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetControlGearStatusRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetControlGearStatusRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetControlGearStatusRequest): GetControlGearStatusRequest.AsObject;
  static serializeBinaryToWriter(message: GetControlGearStatusRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetControlGearStatusRequest;
  static deserializeBinaryFromReader(message: GetControlGearStatusRequest, reader: jspb.BinaryReader): GetControlGearStatusRequest;
}

export namespace GetControlGearStatusRequest {
  export type AsObject = {
    name: string;
  };
}

export class IdentifyRequest extends jspb.Message {
  getName(): string;
  setName(value: string): IdentifyRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): IdentifyRequest.AsObject;
  static toObject(includeInstance: boolean, msg: IdentifyRequest): IdentifyRequest.AsObject;
  static serializeBinaryToWriter(message: IdentifyRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): IdentifyRequest;
  static deserializeBinaryFromReader(message: IdentifyRequest, reader: jspb.BinaryReader): IdentifyRequest;
}

export namespace IdentifyRequest {
  export type AsObject = {
    name: string;
  };
}

export class IdentifyResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): IdentifyResponse.AsObject;
  static toObject(includeInstance: boolean, msg: IdentifyResponse): IdentifyResponse.AsObject;
  static serializeBinaryToWriter(message: IdentifyResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): IdentifyResponse;
  static deserializeBinaryFromReader(message: IdentifyResponse, reader: jspb.BinaryReader): IdentifyResponse;
}

export namespace IdentifyResponse {
  export type AsObject = {
  };
}

export class StartTestRequest extends jspb.Message {
  getName(): string;
  setName(value: string): StartTestRequest;

  getTest(): EmergencyStatus.Test;
  setTest(value: EmergencyStatus.Test): StartTestRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StartTestRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StartTestRequest): StartTestRequest.AsObject;
  static serializeBinaryToWriter(message: StartTestRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StartTestRequest;
  static deserializeBinaryFromReader(message: StartTestRequest, reader: jspb.BinaryReader): StartTestRequest;
}

export namespace StartTestRequest {
  export type AsObject = {
    name: string;
    test: EmergencyStatus.Test;
  };
}

export class StartTestResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StartTestResponse.AsObject;
  static toObject(includeInstance: boolean, msg: StartTestResponse): StartTestResponse.AsObject;
  static serializeBinaryToWriter(message: StartTestResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StartTestResponse;
  static deserializeBinaryFromReader(message: StartTestResponse, reader: jspb.BinaryReader): StartTestResponse;
}

export namespace StartTestResponse {
  export type AsObject = {
  };
}

export class StopTestRequest extends jspb.Message {
  getName(): string;
  setName(value: string): StopTestRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StopTestRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StopTestRequest): StopTestRequest.AsObject;
  static serializeBinaryToWriter(message: StopTestRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StopTestRequest;
  static deserializeBinaryFromReader(message: StopTestRequest, reader: jspb.BinaryReader): StopTestRequest;
}

export namespace StopTestRequest {
  export type AsObject = {
    name: string;
  };
}

export class StopTestResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StopTestResponse.AsObject;
  static toObject(includeInstance: boolean, msg: StopTestResponse): StopTestResponse.AsObject;
  static serializeBinaryToWriter(message: StopTestResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StopTestResponse;
  static deserializeBinaryFromReader(message: StopTestResponse, reader: jspb.BinaryReader): StopTestResponse;
}

export namespace StopTestResponse {
  export type AsObject = {
  };
}

export class UpdateTestIntervalRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateTestIntervalRequest;

  getTest(): EmergencyStatus.Test;
  setTest(value: EmergencyStatus.Test): UpdateTestIntervalRequest;

  getInterval(): google_protobuf_duration_pb.Duration | undefined;
  setInterval(value?: google_protobuf_duration_pb.Duration): UpdateTestIntervalRequest;
  hasInterval(): boolean;
  clearInterval(): UpdateTestIntervalRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateTestIntervalRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateTestIntervalRequest): UpdateTestIntervalRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateTestIntervalRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateTestIntervalRequest;
  static deserializeBinaryFromReader(message: UpdateTestIntervalRequest, reader: jspb.BinaryReader): UpdateTestIntervalRequest;
}

export namespace UpdateTestIntervalRequest {
  export type AsObject = {
    name: string;
    test: EmergencyStatus.Test;
    interval?: google_protobuf_duration_pb.Duration.AsObject;
  };
}

export class UpdateTestIntervalResponse extends jspb.Message {
  getInterval(): google_protobuf_duration_pb.Duration | undefined;
  setInterval(value?: google_protobuf_duration_pb.Duration): UpdateTestIntervalResponse;
  hasInterval(): boolean;
  clearInterval(): UpdateTestIntervalResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateTestIntervalResponse.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateTestIntervalResponse): UpdateTestIntervalResponse.AsObject;
  static serializeBinaryToWriter(message: UpdateTestIntervalResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateTestIntervalResponse;
  static deserializeBinaryFromReader(message: UpdateTestIntervalResponse, reader: jspb.BinaryReader): UpdateTestIntervalResponse;
}

export namespace UpdateTestIntervalResponse {
  export type AsObject = {
    interval?: google_protobuf_duration_pb.Duration.AsObject;
  };
}

export class TestResult extends jspb.Message {
  getTest(): EmergencyStatus.Test;
  setTest(value: EmergencyStatus.Test): TestResult;

  getPass(): boolean;
  setPass(value: boolean): TestResult;

  getDuration(): google_protobuf_duration_pb.Duration | undefined;
  setDuration(value?: google_protobuf_duration_pb.Duration): TestResult;
  hasDuration(): boolean;
  clearDuration(): TestResult;

  getEtag(): string;
  setEtag(value: string): TestResult;

  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): TestResult;
  hasStartTime(): boolean;
  clearStartTime(): TestResult;

  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): TestResult;
  hasEndTime(): boolean;
  clearEndTime(): TestResult;

  getFailureReason(): EmergencyStatus.Failure;
  setFailureReason(value: EmergencyStatus.Failure): TestResult;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TestResult.AsObject;
  static toObject(includeInstance: boolean, msg: TestResult): TestResult.AsObject;
  static serializeBinaryToWriter(message: TestResult, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TestResult;
  static deserializeBinaryFromReader(message: TestResult, reader: jspb.BinaryReader): TestResult;
}

export namespace TestResult {
  export type AsObject = {
    test: EmergencyStatus.Test;
    pass: boolean;
    duration?: google_protobuf_duration_pb.Duration.AsObject;
    etag: string;
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    failureReason: EmergencyStatus.Failure;
  };
}

export class GetTestResultRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetTestResultRequest;

  getTest(): EmergencyStatus.Test;
  setTest(value: EmergencyStatus.Test): GetTestResultRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetTestResultRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetTestResultRequest): GetTestResultRequest.AsObject;
  static serializeBinaryToWriter(message: GetTestResultRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetTestResultRequest;
  static deserializeBinaryFromReader(message: GetTestResultRequest, reader: jspb.BinaryReader): GetTestResultRequest;
}

export namespace GetTestResultRequest {
  export type AsObject = {
    name: string;
    test: EmergencyStatus.Test;
  };
}

export class DeleteTestResultRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DeleteTestResultRequest;

  getTest(): EmergencyStatus.Test;
  setTest(value: EmergencyStatus.Test): DeleteTestResultRequest;

  getEtag(): string;
  setEtag(value: string): DeleteTestResultRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteTestResultRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteTestResultRequest): DeleteTestResultRequest.AsObject;
  static serializeBinaryToWriter(message: DeleteTestResultRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteTestResultRequest;
  static deserializeBinaryFromReader(message: DeleteTestResultRequest, reader: jspb.BinaryReader): DeleteTestResultRequest;
}

export namespace DeleteTestResultRequest {
  export type AsObject = {
    name: string;
    test: EmergencyStatus.Test;
    etag: string;
  };
}

