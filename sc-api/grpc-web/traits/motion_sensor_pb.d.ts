import * as jspb from 'google-protobuf'

import * as google_protobuf_duration_pb from 'google-protobuf/google/protobuf/duration_pb'; // proto import: "google/protobuf/duration.proto"
import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"


export class MotionDetection extends jspb.Message {
  getState(): MotionDetection.State;
  setState(value: MotionDetection.State): MotionDetection;

  getStateChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStateChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): MotionDetection;
  hasStateChangeTime(): boolean;
  clearStateChangeTime(): MotionDetection;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MotionDetection.AsObject;
  static toObject(includeInstance: boolean, msg: MotionDetection): MotionDetection.AsObject;
  static serializeBinaryToWriter(message: MotionDetection, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MotionDetection;
  static deserializeBinaryFromReader(message: MotionDetection, reader: jspb.BinaryReader): MotionDetection;
}

export namespace MotionDetection {
  export type AsObject = {
    state: MotionDetection.State;
    stateChangeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };

  export enum State {
    STATE_UNSPECIFIED = 0,
    NOT_DETECTED = 1,
    DETECTED = 2,
  }
}

export class MotionDetectionSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): MotionDetectionSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): MotionDetectionSupport;

  getNotDetectedDelay(): google_protobuf_duration_pb.Duration | undefined;
  setNotDetectedDelay(value?: google_protobuf_duration_pb.Duration): MotionDetectionSupport;
  hasNotDetectedDelay(): boolean;
  clearNotDetectedDelay(): MotionDetectionSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MotionDetectionSupport.AsObject;
  static toObject(includeInstance: boolean, msg: MotionDetectionSupport): MotionDetectionSupport.AsObject;
  static serializeBinaryToWriter(message: MotionDetectionSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MotionDetectionSupport;
  static deserializeBinaryFromReader(message: MotionDetectionSupport, reader: jspb.BinaryReader): MotionDetectionSupport;
}

export namespace MotionDetectionSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    notDetectedDelay?: google_protobuf_duration_pb.Duration.AsObject;
  };
}

export class GetMotionDetectionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetMotionDetectionRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetMotionDetectionRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetMotionDetectionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetMotionDetectionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetMotionDetectionRequest): GetMotionDetectionRequest.AsObject;
  static serializeBinaryToWriter(message: GetMotionDetectionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetMotionDetectionRequest;
  static deserializeBinaryFromReader(message: GetMotionDetectionRequest, reader: jspb.BinaryReader): GetMotionDetectionRequest;
}

export namespace GetMotionDetectionRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullMotionDetectionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullMotionDetectionRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullMotionDetectionRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullMotionDetectionRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullMotionDetectionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullMotionDetectionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullMotionDetectionRequest): PullMotionDetectionRequest.AsObject;
  static serializeBinaryToWriter(message: PullMotionDetectionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullMotionDetectionRequest;
  static deserializeBinaryFromReader(message: PullMotionDetectionRequest, reader: jspb.BinaryReader): PullMotionDetectionRequest;
}

export namespace PullMotionDetectionRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullMotionDetectionResponse extends jspb.Message {
  getChangesList(): Array<PullMotionDetectionResponse.Change>;
  setChangesList(value: Array<PullMotionDetectionResponse.Change>): PullMotionDetectionResponse;
  clearChangesList(): PullMotionDetectionResponse;
  addChanges(value?: PullMotionDetectionResponse.Change, index?: number): PullMotionDetectionResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullMotionDetectionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullMotionDetectionResponse): PullMotionDetectionResponse.AsObject;
  static serializeBinaryToWriter(message: PullMotionDetectionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullMotionDetectionResponse;
  static deserializeBinaryFromReader(message: PullMotionDetectionResponse, reader: jspb.BinaryReader): PullMotionDetectionResponse;
}

export namespace PullMotionDetectionResponse {
  export type AsObject = {
    changesList: Array<PullMotionDetectionResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getMotionDetection(): MotionDetection | undefined;
    setMotionDetection(value?: MotionDetection): Change;
    hasMotionDetection(): boolean;
    clearMotionDetection(): Change;

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
      motionDetection?: MotionDetection.AsObject;
    };
  }

}

export class DescribeMotionDetectionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeMotionDetectionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeMotionDetectionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeMotionDetectionRequest): DescribeMotionDetectionRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeMotionDetectionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeMotionDetectionRequest;
  static deserializeBinaryFromReader(message: DescribeMotionDetectionRequest, reader: jspb.BinaryReader): DescribeMotionDetectionRequest;
}

export namespace DescribeMotionDetectionRequest {
  export type AsObject = {
    name: string;
  };
}

