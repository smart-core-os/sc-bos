import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_change_pb from '../types/change_pb'; // proto import: "types/change.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"


export class Hail extends jspb.Message {
  getId(): string;
  setId(value: string): Hail;

  getOrigin(): Hail.Location | undefined;
  setOrigin(value?: Hail.Location): Hail;
  hasOrigin(): boolean;
  clearOrigin(): Hail;

  getDestination(): Hail.Location | undefined;
  setDestination(value?: Hail.Location): Hail;
  hasDestination(): boolean;
  clearDestination(): Hail;

  getState(): Hail.State;
  setState(value: Hail.State): Hail;

  getCallTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCallTime(value?: google_protobuf_timestamp_pb.Timestamp): Hail;
  hasCallTime(): boolean;
  clearCallTime(): Hail;

  getBoardTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setBoardTime(value?: google_protobuf_timestamp_pb.Timestamp): Hail;
  hasBoardTime(): boolean;
  clearBoardTime(): Hail;

  getDepartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setDepartTime(value?: google_protobuf_timestamp_pb.Timestamp): Hail;
  hasDepartTime(): boolean;
  clearDepartTime(): Hail;

  getArriveTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setArriveTime(value?: google_protobuf_timestamp_pb.Timestamp): Hail;
  hasArriveTime(): boolean;
  clearArriveTime(): Hail;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Hail.AsObject;
  static toObject(includeInstance: boolean, msg: Hail): Hail.AsObject;
  static serializeBinaryToWriter(message: Hail, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Hail;
  static deserializeBinaryFromReader(message: Hail, reader: jspb.BinaryReader): Hail;
}

export namespace Hail {
  export type AsObject = {
    id: string;
    origin?: Hail.Location.AsObject;
    destination?: Hail.Location.AsObject;
    state: Hail.State;
    callTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    boardTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    departTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    arriveTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };

  export class Location extends jspb.Message {
    getName(): string;
    setName(value: string): Location;

    getDisplayName(): string;
    setDisplayName(value: string): Location;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Location.AsObject;
    static toObject(includeInstance: boolean, msg: Location): Location.AsObject;
    static serializeBinaryToWriter(message: Location, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Location;
    static deserializeBinaryFromReader(message: Location, reader: jspb.BinaryReader): Location;
  }

  export namespace Location {
    export type AsObject = {
      name: string;
      displayName: string;
    };
  }


  export enum State {
    STATE_UNSPECIFIED = 0,
    CALLED = 1,
    BOARDING = 2,
    DEPARTED = 3,
    ARRIVED = 4,
  }
}

export class CreateHailRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CreateHailRequest;

  getHail(): Hail | undefined;
  setHail(value?: Hail): CreateHailRequest;
  hasHail(): boolean;
  clearHail(): CreateHailRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateHailRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateHailRequest): CreateHailRequest.AsObject;
  static serializeBinaryToWriter(message: CreateHailRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateHailRequest;
  static deserializeBinaryFromReader(message: CreateHailRequest, reader: jspb.BinaryReader): CreateHailRequest;
}

export namespace CreateHailRequest {
  export type AsObject = {
    name: string;
    hail?: Hail.AsObject;
  };
}

export class GetHailRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetHailRequest;

  getId(): string;
  setId(value: string): GetHailRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetHailRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetHailRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetHailRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetHailRequest): GetHailRequest.AsObject;
  static serializeBinaryToWriter(message: GetHailRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetHailRequest;
  static deserializeBinaryFromReader(message: GetHailRequest, reader: jspb.BinaryReader): GetHailRequest;
}

export namespace GetHailRequest {
  export type AsObject = {
    name: string;
    id: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateHailRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateHailRequest;

  getHail(): Hail | undefined;
  setHail(value?: Hail): UpdateHailRequest;
  hasHail(): boolean;
  clearHail(): UpdateHailRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateHailRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateHailRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateHailRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateHailRequest): UpdateHailRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateHailRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateHailRequest;
  static deserializeBinaryFromReader(message: UpdateHailRequest, reader: jspb.BinaryReader): UpdateHailRequest;
}

export namespace UpdateHailRequest {
  export type AsObject = {
    name: string;
    hail?: Hail.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class DeleteHailRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DeleteHailRequest;

  getId(): string;
  setId(value: string): DeleteHailRequest;

  getAllowMissing(): boolean;
  setAllowMissing(value: boolean): DeleteHailRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteHailRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteHailRequest): DeleteHailRequest.AsObject;
  static serializeBinaryToWriter(message: DeleteHailRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteHailRequest;
  static deserializeBinaryFromReader(message: DeleteHailRequest, reader: jspb.BinaryReader): DeleteHailRequest;
}

export namespace DeleteHailRequest {
  export type AsObject = {
    name: string;
    id: string;
    allowMissing: boolean;
  };
}

export class DeleteHailResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteHailResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteHailResponse): DeleteHailResponse.AsObject;
  static serializeBinaryToWriter(message: DeleteHailResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteHailResponse;
  static deserializeBinaryFromReader(message: DeleteHailResponse, reader: jspb.BinaryReader): DeleteHailResponse;
}

export namespace DeleteHailResponse {
  export type AsObject = {
  };
}

export class PullHailRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullHailRequest;

  getId(): string;
  setId(value: string): PullHailRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullHailRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullHailRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullHailRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullHailRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullHailRequest): PullHailRequest.AsObject;
  static serializeBinaryToWriter(message: PullHailRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullHailRequest;
  static deserializeBinaryFromReader(message: PullHailRequest, reader: jspb.BinaryReader): PullHailRequest;
}

export namespace PullHailRequest {
  export type AsObject = {
    name: string;
    id: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullHailResponse extends jspb.Message {
  getChangesList(): Array<PullHailResponse.Change>;
  setChangesList(value: Array<PullHailResponse.Change>): PullHailResponse;
  clearChangesList(): PullHailResponse;
  addChanges(value?: PullHailResponse.Change, index?: number): PullHailResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullHailResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullHailResponse): PullHailResponse.AsObject;
  static serializeBinaryToWriter(message: PullHailResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullHailResponse;
  static deserializeBinaryFromReader(message: PullHailResponse, reader: jspb.BinaryReader): PullHailResponse;
}

export namespace PullHailResponse {
  export type AsObject = {
    changesList: Array<PullHailResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getHail(): Hail | undefined;
    setHail(value?: Hail): Change;
    hasHail(): boolean;
    clearHail(): Change;

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
      hail?: Hail.AsObject;
    };
  }

}

export class ListHailsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListHailsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListHailsRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListHailsRequest;

  getPageSize(): number;
  setPageSize(value: number): ListHailsRequest;

  getPageToken(): string;
  setPageToken(value: string): ListHailsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListHailsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListHailsRequest): ListHailsRequest.AsObject;
  static serializeBinaryToWriter(message: ListHailsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListHailsRequest;
  static deserializeBinaryFromReader(message: ListHailsRequest, reader: jspb.BinaryReader): ListHailsRequest;
}

export namespace ListHailsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
  };
}

export class ListHailsResponse extends jspb.Message {
  getHailsList(): Array<Hail>;
  setHailsList(value: Array<Hail>): ListHailsResponse;
  clearHailsList(): ListHailsResponse;
  addHails(value?: Hail, index?: number): Hail;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListHailsResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListHailsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListHailsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListHailsResponse): ListHailsResponse.AsObject;
  static serializeBinaryToWriter(message: ListHailsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListHailsResponse;
  static deserializeBinaryFromReader(message: ListHailsResponse, reader: jspb.BinaryReader): ListHailsResponse;
}

export namespace ListHailsResponse {
  export type AsObject = {
    hailsList: Array<Hail.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class PullHailsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullHailsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullHailsRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullHailsRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullHailsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullHailsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullHailsRequest): PullHailsRequest.AsObject;
  static serializeBinaryToWriter(message: PullHailsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullHailsRequest;
  static deserializeBinaryFromReader(message: PullHailsRequest, reader: jspb.BinaryReader): PullHailsRequest;
}

export namespace PullHailsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullHailsResponse extends jspb.Message {
  getChangesList(): Array<PullHailsResponse.Change>;
  setChangesList(value: Array<PullHailsResponse.Change>): PullHailsResponse;
  clearChangesList(): PullHailsResponse;
  addChanges(value?: PullHailsResponse.Change, index?: number): PullHailsResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullHailsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullHailsResponse): PullHailsResponse.AsObject;
  static serializeBinaryToWriter(message: PullHailsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullHailsResponse;
  static deserializeBinaryFromReader(message: PullHailsResponse, reader: jspb.BinaryReader): PullHailsResponse;
}

export namespace PullHailsResponse {
  export type AsObject = {
    changesList: Array<PullHailsResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getType(): types_change_pb.ChangeType;
    setType(value: types_change_pb.ChangeType): Change;

    getNewValue(): Hail | undefined;
    setNewValue(value?: Hail): Change;
    hasNewValue(): boolean;
    clearNewValue(): Change;

    getOldValue(): Hail | undefined;
    setOldValue(value?: Hail): Change;
    hasOldValue(): boolean;
    clearOldValue(): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

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
      type: types_change_pb.ChangeType;
      newValue?: Hail.AsObject;
      oldValue?: Hail.AsObject;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };
  }

}

export class DescribeHailRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeHailRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeHailRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeHailRequest): DescribeHailRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeHailRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeHailRequest;
  static deserializeBinaryFromReader(message: DescribeHailRequest, reader: jspb.BinaryReader): DescribeHailRequest;
}

export namespace DescribeHailRequest {
  export type AsObject = {
    name: string;
  };
}

export class HailSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): HailSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): HailSupport;

  getSupportedLocationsList(): Array<Hail.Location>;
  setSupportedLocationsList(value: Array<Hail.Location>): HailSupport;
  clearSupportedLocationsList(): HailSupport;
  addSupportedLocations(value?: Hail.Location, index?: number): Hail.Location;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): HailSupport.AsObject;
  static toObject(includeInstance: boolean, msg: HailSupport): HailSupport.AsObject;
  static serializeBinaryToWriter(message: HailSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): HailSupport;
  static deserializeBinaryFromReader(message: HailSupport, reader: jspb.BinaryReader): HailSupport;
}

export namespace HailSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    supportedLocationsList: Array<Hail.Location.AsObject>;
  };
}

