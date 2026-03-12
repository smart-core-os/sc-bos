import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_actor_v1_actor_pb from '../../../../smartcore/bos/actor/v1/actor_pb'; // proto import: "smartcore/bos/actor/v1/actor.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"


export class RebootEvent extends jspb.Message {
  getRebootTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setRebootTime(value?: google_protobuf_timestamp_pb.Timestamp): RebootEvent;
  hasRebootTime(): boolean;
  clearRebootTime(): RebootEvent;

  getReason(): string;
  setReason(value: string): RebootEvent;

  getActor(): smartcore_bos_actor_v1_actor_pb.Actor | undefined;
  setActor(value?: smartcore_bos_actor_v1_actor_pb.Actor): RebootEvent;
  hasActor(): boolean;
  clearActor(): RebootEvent;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RebootEvent.AsObject;
  static toObject(includeInstance: boolean, msg: RebootEvent): RebootEvent.AsObject;
  static serializeBinaryToWriter(message: RebootEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RebootEvent;
  static deserializeBinaryFromReader(message: RebootEvent, reader: jspb.BinaryReader): RebootEvent;
}

export namespace RebootEvent {
  export type AsObject = {
    rebootTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    reason: string;
    actor?: smartcore_bos_actor_v1_actor_pb.Actor.AsObject;
  };
}

export class ListRebootEventsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListRebootEventsRequest;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): ListRebootEventsRequest;
  hasPeriod(): boolean;
  clearPeriod(): ListRebootEventsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListRebootEventsRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListRebootEventsRequest;

  getPageSize(): number;
  setPageSize(value: number): ListRebootEventsRequest;

  getPageToken(): string;
  setPageToken(value: string): ListRebootEventsRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListRebootEventsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListRebootEventsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListRebootEventsRequest): ListRebootEventsRequest.AsObject;
  static serializeBinaryToWriter(message: ListRebootEventsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListRebootEventsRequest;
  static deserializeBinaryFromReader(message: ListRebootEventsRequest, reader: jspb.BinaryReader): ListRebootEventsRequest;
}

export namespace ListRebootEventsRequest {
  export type AsObject = {
    name: string;
    period?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
    orderBy: string;
  };
}

export class ListRebootEventsResponse extends jspb.Message {
  getRebootEventsList(): Array<RebootEvent>;
  setRebootEventsList(value: Array<RebootEvent>): ListRebootEventsResponse;
  clearRebootEventsList(): ListRebootEventsResponse;
  addRebootEvents(value?: RebootEvent, index?: number): RebootEvent;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListRebootEventsResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListRebootEventsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListRebootEventsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListRebootEventsResponse): ListRebootEventsResponse.AsObject;
  static serializeBinaryToWriter(message: ListRebootEventsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListRebootEventsResponse;
  static deserializeBinaryFromReader(message: ListRebootEventsResponse, reader: jspb.BinaryReader): ListRebootEventsResponse;
}

export namespace ListRebootEventsResponse {
  export type AsObject = {
    rebootEventsList: Array<RebootEvent.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

