import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_actor_v1_actor_pb from '../../../../smartcore/bos/actor/v1/actor_pb'; // proto import: "smartcore/bos/actor/v1/actor.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"


export class BootEvent extends jspb.Message {
  getRebootTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRebootTime(value?: google_protobuf_timestamp_pb.Timestamp): BootEvent;
  hasRebootTime(): boolean;
  clearRebootTime(): BootEvent;

  getReason(): string;
  setReason(value: string): BootEvent;

  getActor(): smartcore_bos_actor_v1_actor_pb.Actor | undefined;
  setActor(value?: smartcore_bos_actor_v1_actor_pb.Actor): BootEvent;
  hasActor(): boolean;
  clearActor(): BootEvent;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BootEvent.AsObject;
  static toObject(includeInstance: boolean, msg: BootEvent): BootEvent.AsObject;
  static serializeBinaryToWriter(message: BootEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BootEvent;
  static deserializeBinaryFromReader(message: BootEvent, reader: jspb.BinaryReader): BootEvent;
}

export namespace BootEvent {
  export type AsObject = {
    rebootTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    reason: string;
    actor?: smartcore_bos_actor_v1_actor_pb.Actor.AsObject;
  };
}

export class ListBootEventsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListBootEventsRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListBootEventsRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListBootEventsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListBootEventsRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListBootEventsRequest;

  getPageSize(): number;
  setPageSize(value: number): ListBootEventsRequest;

  getPageToken(): string;
  setPageToken(value: string): ListBootEventsRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListBootEventsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListBootEventsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListBootEventsRequest): ListBootEventsRequest.AsObject;
  static serializeBinaryToWriter(message: ListBootEventsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListBootEventsRequest;
  static deserializeBinaryFromReader(message: ListBootEventsRequest, reader: jspb.BinaryReader): ListBootEventsRequest;
}

export namespace ListBootEventsRequest {
  export type AsObject = {
    name: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListBootEventsResponse extends jspb.Message {
  getBootEventsList(): Array<BootEvent>;
  setBootEventsList(value: Array<BootEvent>): ListBootEventsResponse;
  clearBootEventsList(): ListBootEventsResponse;
  addBootEvents(value?: BootEvent, index?: number): BootEvent;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListBootEventsResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListBootEventsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListBootEventsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListBootEventsResponse): ListBootEventsResponse.AsObject;
  static serializeBinaryToWriter(message: ListBootEventsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListBootEventsResponse;
  static deserializeBinaryFromReader(message: ListBootEventsResponse, reader: jspb.BinaryReader): ListBootEventsResponse;
}

export namespace ListBootEventsResponse {
  export type AsObject = {
    bootEventsList: Array<BootEvent.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

