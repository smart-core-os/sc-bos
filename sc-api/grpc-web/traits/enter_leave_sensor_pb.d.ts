import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_image_pb from '../types/image_pb'; // proto import: "types/image.proto"


export class EnterLeaveEvent extends jspb.Message {
  getDirection(): EnterLeaveEvent.Direction;
  setDirection(value: EnterLeaveEvent.Direction): EnterLeaveEvent;

  getOccupant(): EnterLeaveEvent.Occupant | undefined;
  setOccupant(value?: EnterLeaveEvent.Occupant): EnterLeaveEvent;
  hasOccupant(): boolean;
  clearOccupant(): EnterLeaveEvent;

  getEnterTotal(): number;
  setEnterTotal(value: number): EnterLeaveEvent;
  hasEnterTotal(): boolean;
  clearEnterTotal(): EnterLeaveEvent;

  getLeaveTotal(): number;
  setLeaveTotal(value: number): EnterLeaveEvent;
  hasLeaveTotal(): boolean;
  clearLeaveTotal(): EnterLeaveEvent;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): EnterLeaveEvent.AsObject;
  static toObject(includeInstance: boolean, msg: EnterLeaveEvent): EnterLeaveEvent.AsObject;
  static serializeBinaryToWriter(message: EnterLeaveEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): EnterLeaveEvent;
  static deserializeBinaryFromReader(message: EnterLeaveEvent, reader: jspb.BinaryReader): EnterLeaveEvent;
}

export namespace EnterLeaveEvent {
  export type AsObject = {
    direction: EnterLeaveEvent.Direction;
    occupant?: EnterLeaveEvent.Occupant.AsObject;
    enterTotal?: number;
    leaveTotal?: number;
  };

  export class Occupant extends jspb.Message {
    getName(): string;
    setName(value: string): Occupant;

    getTitle(): string;
    setTitle(value: string): Occupant;

    getDisplayName(): string;
    setDisplayName(value: string): Occupant;

    getPicture(): types_image_pb.Image | undefined;
    setPicture(value?: types_image_pb.Image): Occupant;
    hasPicture(): boolean;
    clearPicture(): Occupant;

    getUrl(): string;
    setUrl(value: string): Occupant;

    getEmail(): string;
    setEmail(value: string): Occupant;

    getIdsMap(): jspb.Map<string, string>;
    clearIdsMap(): Occupant;

    getMoreMap(): jspb.Map<string, string>;
    clearMoreMap(): Occupant;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Occupant.AsObject;
    static toObject(includeInstance: boolean, msg: Occupant): Occupant.AsObject;
    static serializeBinaryToWriter(message: Occupant, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Occupant;
    static deserializeBinaryFromReader(message: Occupant, reader: jspb.BinaryReader): Occupant;
  }

  export namespace Occupant {
    export type AsObject = {
      name: string;
      title: string;
      displayName: string;
      picture?: types_image_pb.Image.AsObject;
      url: string;
      email: string;
      idsMap: Array<[string, string]>;
      moreMap: Array<[string, string]>;
    };
  }


  export enum Direction {
    DIRECTION_UNSPECIFIED = 0,
    ENTER = 1,
    LEAVE = 2,
  }

  export enum EnterTotalCase {
    _ENTER_TOTAL_NOT_SET = 0,
    ENTER_TOTAL = 3,
  }

  export enum LeaveTotalCase {
    _LEAVE_TOTAL_NOT_SET = 0,
    LEAVE_TOTAL = 4,
  }
}

export class PullEnterLeaveEventsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullEnterLeaveEventsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullEnterLeaveEventsRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullEnterLeaveEventsRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullEnterLeaveEventsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullEnterLeaveEventsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullEnterLeaveEventsRequest): PullEnterLeaveEventsRequest.AsObject;
  static serializeBinaryToWriter(message: PullEnterLeaveEventsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullEnterLeaveEventsRequest;
  static deserializeBinaryFromReader(message: PullEnterLeaveEventsRequest, reader: jspb.BinaryReader): PullEnterLeaveEventsRequest;
}

export namespace PullEnterLeaveEventsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullEnterLeaveEventsResponse extends jspb.Message {
  getChangesList(): Array<PullEnterLeaveEventsResponse.Change>;
  setChangesList(value: Array<PullEnterLeaveEventsResponse.Change>): PullEnterLeaveEventsResponse;
  clearChangesList(): PullEnterLeaveEventsResponse;
  addChanges(value?: PullEnterLeaveEventsResponse.Change, index?: number): PullEnterLeaveEventsResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullEnterLeaveEventsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullEnterLeaveEventsResponse): PullEnterLeaveEventsResponse.AsObject;
  static serializeBinaryToWriter(message: PullEnterLeaveEventsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullEnterLeaveEventsResponse;
  static deserializeBinaryFromReader(message: PullEnterLeaveEventsResponse, reader: jspb.BinaryReader): PullEnterLeaveEventsResponse;
}

export namespace PullEnterLeaveEventsResponse {
  export type AsObject = {
    changesList: Array<PullEnterLeaveEventsResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getEnterLeaveEvent(): EnterLeaveEvent | undefined;
    setEnterLeaveEvent(value?: EnterLeaveEvent): Change;
    hasEnterLeaveEvent(): boolean;
    clearEnterLeaveEvent(): Change;

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
      enterLeaveEvent?: EnterLeaveEvent.AsObject;
    };
  }

}

export class GetEnterLeaveEventRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetEnterLeaveEventRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetEnterLeaveEventRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetEnterLeaveEventRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetEnterLeaveEventRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetEnterLeaveEventRequest): GetEnterLeaveEventRequest.AsObject;
  static serializeBinaryToWriter(message: GetEnterLeaveEventRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetEnterLeaveEventRequest;
  static deserializeBinaryFromReader(message: GetEnterLeaveEventRequest, reader: jspb.BinaryReader): GetEnterLeaveEventRequest;
}

export namespace GetEnterLeaveEventRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class ResetEnterLeaveTotalsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ResetEnterLeaveTotalsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ResetEnterLeaveTotalsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ResetEnterLeaveTotalsRequest): ResetEnterLeaveTotalsRequest.AsObject;
  static serializeBinaryToWriter(message: ResetEnterLeaveTotalsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ResetEnterLeaveTotalsRequest;
  static deserializeBinaryFromReader(message: ResetEnterLeaveTotalsRequest, reader: jspb.BinaryReader): ResetEnterLeaveTotalsRequest;
}

export namespace ResetEnterLeaveTotalsRequest {
  export type AsObject = {
    name: string;
  };
}

export class ResetEnterLeaveTotalsResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ResetEnterLeaveTotalsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ResetEnterLeaveTotalsResponse): ResetEnterLeaveTotalsResponse.AsObject;
  static serializeBinaryToWriter(message: ResetEnterLeaveTotalsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ResetEnterLeaveTotalsResponse;
  static deserializeBinaryFromReader(message: ResetEnterLeaveTotalsResponse, reader: jspb.BinaryReader): ResetEnterLeaveTotalsResponse;
}

export namespace ResetEnterLeaveTotalsResponse {
  export type AsObject = {
  };
}

