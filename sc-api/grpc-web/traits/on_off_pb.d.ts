import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"


export class OnOff extends jspb.Message {
  getState(): OnOff.State;
  setState(value: OnOff.State): OnOff;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): OnOff.AsObject;
  static toObject(includeInstance: boolean, msg: OnOff): OnOff.AsObject;
  static serializeBinaryToWriter(message: OnOff, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): OnOff;
  static deserializeBinaryFromReader(message: OnOff, reader: jspb.BinaryReader): OnOff;
}

export namespace OnOff {
  export type AsObject = {
    state: OnOff.State;
  };

  export enum State {
    STATE_UNSPECIFIED = 0,
    ON = 1,
    OFF = 2,
  }
}

export class OnOffSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): OnOffSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): OnOffSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): OnOffSupport.AsObject;
  static toObject(includeInstance: boolean, msg: OnOffSupport): OnOffSupport.AsObject;
  static serializeBinaryToWriter(message: OnOffSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): OnOffSupport;
  static deserializeBinaryFromReader(message: OnOffSupport, reader: jspb.BinaryReader): OnOffSupport;
}

export namespace OnOffSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
  };
}

export class GetOnOffRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetOnOffRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetOnOffRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetOnOffRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetOnOffRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetOnOffRequest): GetOnOffRequest.AsObject;
  static serializeBinaryToWriter(message: GetOnOffRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetOnOffRequest;
  static deserializeBinaryFromReader(message: GetOnOffRequest, reader: jspb.BinaryReader): GetOnOffRequest;
}

export namespace GetOnOffRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateOnOffRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateOnOffRequest;

  getOnOff(): OnOff | undefined;
  setOnOff(value?: OnOff): UpdateOnOffRequest;
  hasOnOff(): boolean;
  clearOnOff(): UpdateOnOffRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateOnOffRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateOnOffRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateOnOffRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateOnOffRequest): UpdateOnOffRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateOnOffRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateOnOffRequest;
  static deserializeBinaryFromReader(message: UpdateOnOffRequest, reader: jspb.BinaryReader): UpdateOnOffRequest;
}

export namespace UpdateOnOffRequest {
  export type AsObject = {
    name: string;
    onOff?: OnOff.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullOnOffRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullOnOffRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullOnOffRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullOnOffRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullOnOffRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullOnOffRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullOnOffRequest): PullOnOffRequest.AsObject;
  static serializeBinaryToWriter(message: PullOnOffRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullOnOffRequest;
  static deserializeBinaryFromReader(message: PullOnOffRequest, reader: jspb.BinaryReader): PullOnOffRequest;
}

export namespace PullOnOffRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullOnOffResponse extends jspb.Message {
  getChangesList(): Array<PullOnOffResponse.Change>;
  setChangesList(value: Array<PullOnOffResponse.Change>): PullOnOffResponse;
  clearChangesList(): PullOnOffResponse;
  addChanges(value?: PullOnOffResponse.Change, index?: number): PullOnOffResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullOnOffResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullOnOffResponse): PullOnOffResponse.AsObject;
  static serializeBinaryToWriter(message: PullOnOffResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullOnOffResponse;
  static deserializeBinaryFromReader(message: PullOnOffResponse, reader: jspb.BinaryReader): PullOnOffResponse;
}

export namespace PullOnOffResponse {
  export type AsObject = {
    changesList: Array<PullOnOffResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getOnOff(): OnOff | undefined;
    setOnOff(value?: OnOff): Change;
    hasOnOff(): boolean;
    clearOnOff(): Change;

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
      onOff?: OnOff.AsObject;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };
  }

}

export class DescribeOnOffRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeOnOffRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeOnOffRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeOnOffRequest): DescribeOnOffRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeOnOffRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeOnOffRequest;
  static deserializeBinaryFromReader(message: DescribeOnOffRequest, reader: jspb.BinaryReader): DescribeOnOffRequest;
}

export namespace DescribeOnOffRequest {
  export type AsObject = {
    name: string;
  };
}

