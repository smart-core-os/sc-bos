import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_actor_v1_actor_pb from '../../../../smartcore/bos/actor/v1/actor_pb'; // proto import: "smartcore/bos/actor/v1/actor.proto"


export class Allocation extends jspb.Message {
  getState(): Allocation.State;
  setState(value: Allocation.State): Allocation;

  getActor(): smartcore_bos_actor_v1_actor_pb.Actor | undefined;
  setActor(value?: smartcore_bos_actor_v1_actor_pb.Actor): Allocation;
  hasActor(): boolean;
  clearActor(): Allocation;

  getGroupId(): string;
  setGroupId(value: string): Allocation;
  hasGroupId(): boolean;
  clearGroupId(): Allocation;

  getUnallocationTotal(): number;
  setUnallocationTotal(value: number): Allocation;
  hasUnallocationTotal(): boolean;
  clearUnallocationTotal(): Allocation;

  getAllocationTotal(): number;
  setAllocationTotal(value: number): Allocation;
  hasAllocationTotal(): boolean;
  clearAllocationTotal(): Allocation;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Allocation.AsObject;
  static toObject(includeInstance: boolean, msg: Allocation): Allocation.AsObject;
  static serializeBinaryToWriter(message: Allocation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Allocation;
  static deserializeBinaryFromReader(message: Allocation, reader: jspb.BinaryReader): Allocation;
}

export namespace Allocation {
  export type AsObject = {
    state: Allocation.State;
    actor?: smartcore_bos_actor_v1_actor_pb.Actor.AsObject;
    groupId?: string;
    unallocationTotal?: number;
    allocationTotal?: number;
  };

  export enum State {
    STATE_UNSPECIFIED = 0,
    UNALLOCATED = 1,
    ALLOCATED = 2,
  }

  export enum ActorCase {
    _ACTOR_NOT_SET = 0,
    ACTOR = 2,
  }

  export enum GroupIdCase {
    _GROUP_ID_NOT_SET = 0,
    GROUP_ID = 3,
  }

  export enum UnallocationTotalCase {
    _UNALLOCATION_TOTAL_NOT_SET = 0,
    UNALLOCATION_TOTAL = 4,
  }

  export enum AllocationTotalCase {
    _ALLOCATION_TOTAL_NOT_SET = 0,
    ALLOCATION_TOTAL = 5,
  }
}

export class GetAllocationRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetAllocationRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetAllocationRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetAllocationRequest;

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

export class PullAllocationRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullAllocationRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullAllocationRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullAllocationRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullAllocationRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullAllocationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullAllocationRequest): PullAllocationRequest.AsObject;
  static serializeBinaryToWriter(message: PullAllocationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullAllocationRequest;
  static deserializeBinaryFromReader(message: PullAllocationRequest, reader: jspb.BinaryReader): PullAllocationRequest;
}

export namespace PullAllocationRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullAllocationResponse extends jspb.Message {
  getChangesList(): Array<PullAllocationResponse.Change>;
  setChangesList(value: Array<PullAllocationResponse.Change>): PullAllocationResponse;
  clearChangesList(): PullAllocationResponse;
  addChanges(value?: PullAllocationResponse.Change, index?: number): PullAllocationResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullAllocationResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullAllocationResponse): PullAllocationResponse.AsObject;
  static serializeBinaryToWriter(message: PullAllocationResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullAllocationResponse;
  static deserializeBinaryFromReader(message: PullAllocationResponse, reader: jspb.BinaryReader): PullAllocationResponse;
}

export namespace PullAllocationResponse {
  export type AsObject = {
    changesList: Array<PullAllocationResponse.Change.AsObject>;
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

