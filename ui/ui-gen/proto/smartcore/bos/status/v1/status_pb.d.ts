import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class StatusLog extends jspb.Message {
  getLevel(): StatusLog.Level;
  setLevel(value: StatusLog.Level): StatusLog;

  getDescription(): string;
  setDescription(value: string): StatusLog;

  getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): StatusLog;
  hasRecordTime(): boolean;
  clearRecordTime(): StatusLog;

  getProblemsList(): Array<StatusLog.Problem>;
  setProblemsList(value: Array<StatusLog.Problem>): StatusLog;
  clearProblemsList(): StatusLog;
  addProblems(value?: StatusLog.Problem, index?: number): StatusLog.Problem;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StatusLog.AsObject;
  static toObject(includeInstance: boolean, msg: StatusLog): StatusLog.AsObject;
  static serializeBinaryToWriter(message: StatusLog, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StatusLog;
  static deserializeBinaryFromReader(message: StatusLog, reader: jspb.BinaryReader): StatusLog;
}

export namespace StatusLog {
  export type AsObject = {
    level: StatusLog.Level;
    description: string;
    recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    problemsList: Array<StatusLog.Problem.AsObject>;
  };

  export class Problem extends jspb.Message {
    getLevel(): StatusLog.Level;
    setLevel(value: StatusLog.Level): Problem;

    getDescription(): string;
    setDescription(value: string): Problem;

    getRecordTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setRecordTime(value?: google_protobuf_timestamp_pb.Timestamp): Problem;
    hasRecordTime(): boolean;
    clearRecordTime(): Problem;

    getName(): string;
    setName(value: string): Problem;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Problem.AsObject;
    static toObject(includeInstance: boolean, msg: Problem): Problem.AsObject;
    static serializeBinaryToWriter(message: Problem, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Problem;
    static deserializeBinaryFromReader(message: Problem, reader: jspb.BinaryReader): Problem;
  }

  export namespace Problem {
    export type AsObject = {
      level: StatusLog.Level;
      description: string;
      recordTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      name: string;
    };
  }


  export enum Level {
    LEVEL_UNDEFINED = 0,
    NOMINAL = 1,
    NOTICE = 2,
    REDUCED_FUNCTION = 3,
    NON_FUNCTIONAL = 4,
    OFFLINE = 127,
  }
}

export class GetCurrentStatusRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetCurrentStatusRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetCurrentStatusRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetCurrentStatusRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetCurrentStatusRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetCurrentStatusRequest): GetCurrentStatusRequest.AsObject;
  static serializeBinaryToWriter(message: GetCurrentStatusRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetCurrentStatusRequest;
  static deserializeBinaryFromReader(message: GetCurrentStatusRequest, reader: jspb.BinaryReader): GetCurrentStatusRequest;
}

export namespace GetCurrentStatusRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullCurrentStatusRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullCurrentStatusRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullCurrentStatusRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullCurrentStatusRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullCurrentStatusRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullCurrentStatusRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullCurrentStatusRequest): PullCurrentStatusRequest.AsObject;
  static serializeBinaryToWriter(message: PullCurrentStatusRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullCurrentStatusRequest;
  static deserializeBinaryFromReader(message: PullCurrentStatusRequest, reader: jspb.BinaryReader): PullCurrentStatusRequest;
}

export namespace PullCurrentStatusRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullCurrentStatusResponse extends jspb.Message {
  getChangesList(): Array<PullCurrentStatusResponse.Change>;
  setChangesList(value: Array<PullCurrentStatusResponse.Change>): PullCurrentStatusResponse;
  clearChangesList(): PullCurrentStatusResponse;
  addChanges(value?: PullCurrentStatusResponse.Change, index?: number): PullCurrentStatusResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullCurrentStatusResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullCurrentStatusResponse): PullCurrentStatusResponse.AsObject;
  static serializeBinaryToWriter(message: PullCurrentStatusResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullCurrentStatusResponse;
  static deserializeBinaryFromReader(message: PullCurrentStatusResponse, reader: jspb.BinaryReader): PullCurrentStatusResponse;
}

export namespace PullCurrentStatusResponse {
  export type AsObject = {
    changesList: Array<PullCurrentStatusResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getCurrentStatus(): StatusLog | undefined;
    setCurrentStatus(value?: StatusLog): Change;
    hasCurrentStatus(): boolean;
    clearCurrentStatus(): Change;

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
      currentStatus?: StatusLog.AsObject;
    };
  }

}

