import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"


export class Occupancy extends jspb.Message {
  getState(): Occupancy.State;
  setState(value: Occupancy.State): Occupancy;

  getPeopleCount(): number;
  setPeopleCount(value: number): Occupancy;

  getStateChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStateChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Occupancy;
  hasStateChangeTime(): boolean;
  clearStateChangeTime(): Occupancy;

  getReasonsList(): Array<string>;
  setReasonsList(value: Array<string>): Occupancy;
  clearReasonsList(): Occupancy;
  addReasons(value: string, index?: number): Occupancy;

  getConfidence(): number;
  setConfidence(value: number): Occupancy;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Occupancy.AsObject;
  static toObject(includeInstance: boolean, msg: Occupancy): Occupancy.AsObject;
  static serializeBinaryToWriter(message: Occupancy, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Occupancy;
  static deserializeBinaryFromReader(message: Occupancy, reader: jspb.BinaryReader): Occupancy;
}

export namespace Occupancy {
  export type AsObject = {
    state: Occupancy.State;
    peopleCount: number;
    stateChangeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    reasonsList: Array<string>;
    confidence: number;
  };

  export enum State {
    STATE_UNSPECIFIED = 0,
    OCCUPIED = 1,
    UNOCCUPIED = 2,
    IDLE = 3,
  }
}

export class OccupancySupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): OccupancySupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): OccupancySupport;

  getMaxPeople(): number;
  setMaxPeople(value: number): OccupancySupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): OccupancySupport.AsObject;
  static toObject(includeInstance: boolean, msg: OccupancySupport): OccupancySupport.AsObject;
  static serializeBinaryToWriter(message: OccupancySupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): OccupancySupport;
  static deserializeBinaryFromReader(message: OccupancySupport, reader: jspb.BinaryReader): OccupancySupport;
}

export namespace OccupancySupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    maxPeople: number;
  };
}

export class GetOccupancyRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetOccupancyRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetOccupancyRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetOccupancyRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetOccupancyRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetOccupancyRequest): GetOccupancyRequest.AsObject;
  static serializeBinaryToWriter(message: GetOccupancyRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetOccupancyRequest;
  static deserializeBinaryFromReader(message: GetOccupancyRequest, reader: jspb.BinaryReader): GetOccupancyRequest;
}

export namespace GetOccupancyRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullOccupancyRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullOccupancyRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullOccupancyRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullOccupancyRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullOccupancyRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullOccupancyRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullOccupancyRequest): PullOccupancyRequest.AsObject;
  static serializeBinaryToWriter(message: PullOccupancyRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullOccupancyRequest;
  static deserializeBinaryFromReader(message: PullOccupancyRequest, reader: jspb.BinaryReader): PullOccupancyRequest;
}

export namespace PullOccupancyRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullOccupancyResponse extends jspb.Message {
  getChangesList(): Array<PullOccupancyResponse.Change>;
  setChangesList(value: Array<PullOccupancyResponse.Change>): PullOccupancyResponse;
  clearChangesList(): PullOccupancyResponse;
  addChanges(value?: PullOccupancyResponse.Change, index?: number): PullOccupancyResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullOccupancyResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullOccupancyResponse): PullOccupancyResponse.AsObject;
  static serializeBinaryToWriter(message: PullOccupancyResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullOccupancyResponse;
  static deserializeBinaryFromReader(message: PullOccupancyResponse, reader: jspb.BinaryReader): PullOccupancyResponse;
}

export namespace PullOccupancyResponse {
  export type AsObject = {
    changesList: Array<PullOccupancyResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getOccupancy(): Occupancy | undefined;
    setOccupancy(value?: Occupancy): Change;
    hasOccupancy(): boolean;
    clearOccupancy(): Change;

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
      occupancy?: Occupancy.AsObject;
    };
  }

}

export class DescribeOccupancyRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeOccupancyRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeOccupancyRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeOccupancyRequest): DescribeOccupancyRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeOccupancyRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeOccupancyRequest;
  static deserializeBinaryFromReader(message: DescribeOccupancyRequest, reader: jspb.BinaryReader): DescribeOccupancyRequest;
}

export namespace DescribeOccupancyRequest {
  export type AsObject = {
    name: string;
  };
}

