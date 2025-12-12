import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"
import * as actor_pb from './actor_pb'; // proto import: "actor.proto"


export class Allocation extends jspb.Message {
  getId(): string;
  setId(value: string): Allocation;

  getAssignment(): Allocation.Assignment;
  setAssignment(value: Allocation.Assignment): Allocation;

  getActor(): actor_pb.Actor | undefined;
  setActor(value?: actor_pb.Actor): Allocation;
  hasActor(): boolean;
  clearActor(): Allocation;

  getPeriod(): types_time_period_pb.Period | undefined;
  setPeriod(value?: types_time_period_pb.Period): Allocation;
  hasPeriod(): boolean;
  clearPeriod(): Allocation;

  getGroupId(): string;
  setGroupId(value: string): Allocation;
  hasGroupId(): boolean;
  clearGroupId(): Allocation;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Allocation.AsObject;
  static toObject(includeInstance: boolean, msg: Allocation): Allocation.AsObject;
  static serializeBinaryToWriter(message: Allocation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Allocation;
  static deserializeBinaryFromReader(message: Allocation, reader: jspb.BinaryReader): Allocation;
}

export namespace Allocation {
  export type AsObject = {
    id: string;
    assignment: Allocation.Assignment;
    actor?: actor_pb.Actor.AsObject;
    period?: types_time_period_pb.Period.AsObject;
    groupId?: string;
  };

  export enum Assignment {
    ASSIGNMENT_UNSPECIFIED = 0,
    UNASSIGNED = 1,
    ASSIGNED = 2,
    RESERVED = 3,
    ALLOCATED = 4,
    BOOKED = 5,
  }

  export enum ActorCase {
    _ACTOR_NOT_SET = 0,
    ACTOR = 3,
  }

  export enum PeriodCase {
    _PERIOD_NOT_SET = 0,
    PERIOD = 4,
  }

  export enum GroupIdCase {
    _GROUP_ID_NOT_SET = 0,
    GROUP_ID = 5,
  }
}

export class GetAllocationRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetAllocationRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetAllocationRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetAllocationRequest;

  getAllocationId(): string;
  setAllocationId(value: string): GetAllocationRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetAllocationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetAllocationRequest): GetAllocationRequest.AsObject;
  static serializeBinaryToWriter(message: GetAllocationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetAllocationRequest;
  static deserializeBinaryFromReader(message: GetAllocationRequest, reader: jspb.BinaryReader): GetAllocationRequest;
}

export namespace GetAllocationRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    allocationId: string;
  };
}

export class UpdateAllocationRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateAllocationRequest;

  getAllocation(): Allocation | undefined;
  setAllocation(value?: Allocation): UpdateAllocationRequest;
  hasAllocation(): boolean;
  clearAllocation(): UpdateAllocationRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateAllocationRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateAllocationRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateAllocationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateAllocationRequest): UpdateAllocationRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateAllocationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateAllocationRequest;
  static deserializeBinaryFromReader(message: UpdateAllocationRequest, reader: jspb.BinaryReader): UpdateAllocationRequest;
}

export namespace UpdateAllocationRequest {
  export type AsObject = {
    name: string;
    allocation?: Allocation.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullAllocationsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullAllocationsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullAllocationsRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullAllocationsRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullAllocationsRequest;

  getAllocationId(): string;
  setAllocationId(value: string): PullAllocationsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullAllocationsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullAllocationsRequest): PullAllocationsRequest.AsObject;
  static serializeBinaryToWriter(message: PullAllocationsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullAllocationsRequest;
  static deserializeBinaryFromReader(message: PullAllocationsRequest, reader: jspb.BinaryReader): PullAllocationsRequest;
}

export namespace PullAllocationsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
    allocationId: string;
  };
}

export class PullAllocationsResponse extends jspb.Message {
  getChangesList(): Array<PullAllocationsResponse.Change>;
  setChangesList(value: Array<PullAllocationsResponse.Change>): PullAllocationsResponse;
  clearChangesList(): PullAllocationsResponse;
  addChanges(value?: PullAllocationsResponse.Change, index?: number): PullAllocationsResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullAllocationsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullAllocationsResponse): PullAllocationsResponse.AsObject;
  static serializeBinaryToWriter(message: PullAllocationsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullAllocationsResponse;
  static deserializeBinaryFromReader(message: PullAllocationsResponse, reader: jspb.BinaryReader): PullAllocationsResponse;
}

export namespace PullAllocationsResponse {
  export type AsObject = {
    changesList: Array<PullAllocationsResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getAllocation(): Allocation | undefined;
    setAllocation(value?: Allocation): Change;
    hasAllocation(): boolean;
    clearAllocation(): Change;

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
      allocation?: Allocation.AsObject;
    };
  }

}

export class ListAllocatableResourcesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListAllocatableResourcesRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListAllocatableResourcesRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListAllocatableResourcesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAllocatableResourcesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListAllocatableResourcesRequest): ListAllocatableResourcesRequest.AsObject;
  static serializeBinaryToWriter(message: ListAllocatableResourcesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAllocatableResourcesRequest;
  static deserializeBinaryFromReader(message: ListAllocatableResourcesRequest, reader: jspb.BinaryReader): ListAllocatableResourcesRequest;
}

export namespace ListAllocatableResourcesRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class ListAllocatableResourcesResponse extends jspb.Message {
  getAllocationsList(): Array<Allocation>;
  setAllocationsList(value: Array<Allocation>): ListAllocatableResourcesResponse;
  clearAllocationsList(): ListAllocatableResourcesResponse;
  addAllocations(value?: Allocation, index?: number): Allocation;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAllocatableResourcesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListAllocatableResourcesResponse): ListAllocatableResourcesResponse.AsObject;
  static serializeBinaryToWriter(message: ListAllocatableResourcesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAllocatableResourcesResponse;
  static deserializeBinaryFromReader(message: ListAllocatableResourcesResponse, reader: jspb.BinaryReader): ListAllocatableResourcesResponse;
}

export namespace ListAllocatableResourcesResponse {
  export type AsObject = {
    allocationsList: Array<Allocation.AsObject>;
  };
}

