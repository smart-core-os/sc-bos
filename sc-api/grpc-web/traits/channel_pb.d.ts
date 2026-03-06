import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"


export class Channel extends jspb.Message {
  getId(): string;
  setId(value: string): Channel;

  getChannelNumber(): string;
  setChannelNumber(value: string): Channel;

  getTitle(): string;
  setTitle(value: string): Channel;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Channel.AsObject;
  static toObject(includeInstance: boolean, msg: Channel): Channel.AsObject;
  static serializeBinaryToWriter(message: Channel, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Channel;
  static deserializeBinaryFromReader(message: Channel, reader: jspb.BinaryReader): Channel;
}

export namespace Channel {
  export type AsObject = {
    id: string;
    channelNumber: string;
    title: string;
  };
}

export class ChosenChannelSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): ChosenChannelSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): ChosenChannelSupport;

  getAdjustMax(): number;
  setAdjustMax(value: number): ChosenChannelSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ChosenChannelSupport.AsObject;
  static toObject(includeInstance: boolean, msg: ChosenChannelSupport): ChosenChannelSupport.AsObject;
  static serializeBinaryToWriter(message: ChosenChannelSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ChosenChannelSupport;
  static deserializeBinaryFromReader(message: ChosenChannelSupport, reader: jspb.BinaryReader): ChosenChannelSupport;
}

export namespace ChosenChannelSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    adjustMax: number;
  };
}

export class GetChosenChannelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetChosenChannelRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetChosenChannelRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetChosenChannelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetChosenChannelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetChosenChannelRequest): GetChosenChannelRequest.AsObject;
  static serializeBinaryToWriter(message: GetChosenChannelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetChosenChannelRequest;
  static deserializeBinaryFromReader(message: GetChosenChannelRequest, reader: jspb.BinaryReader): GetChosenChannelRequest;
}

export namespace GetChosenChannelRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class ChooseChannelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ChooseChannelRequest;

  getChannel(): Channel | undefined;
  setChannel(value?: Channel): ChooseChannelRequest;
  hasChannel(): boolean;
  clearChannel(): ChooseChannelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ChooseChannelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ChooseChannelRequest): ChooseChannelRequest.AsObject;
  static serializeBinaryToWriter(message: ChooseChannelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ChooseChannelRequest;
  static deserializeBinaryFromReader(message: ChooseChannelRequest, reader: jspb.BinaryReader): ChooseChannelRequest;
}

export namespace ChooseChannelRequest {
  export type AsObject = {
    name: string;
    channel?: Channel.AsObject;
  };
}

export class AdjustChannelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): AdjustChannelRequest;

  getAmount(): number;
  setAmount(value: number): AdjustChannelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AdjustChannelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: AdjustChannelRequest): AdjustChannelRequest.AsObject;
  static serializeBinaryToWriter(message: AdjustChannelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AdjustChannelRequest;
  static deserializeBinaryFromReader(message: AdjustChannelRequest, reader: jspb.BinaryReader): AdjustChannelRequest;
}

export namespace AdjustChannelRequest {
  export type AsObject = {
    name: string;
    amount: number;
  };
}

export class ReturnChannelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ReturnChannelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ReturnChannelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ReturnChannelRequest): ReturnChannelRequest.AsObject;
  static serializeBinaryToWriter(message: ReturnChannelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ReturnChannelRequest;
  static deserializeBinaryFromReader(message: ReturnChannelRequest, reader: jspb.BinaryReader): ReturnChannelRequest;
}

export namespace ReturnChannelRequest {
  export type AsObject = {
    name: string;
  };
}

export class PullChosenChannelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullChosenChannelRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullChosenChannelRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullChosenChannelRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullChosenChannelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullChosenChannelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullChosenChannelRequest): PullChosenChannelRequest.AsObject;
  static serializeBinaryToWriter(message: PullChosenChannelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullChosenChannelRequest;
  static deserializeBinaryFromReader(message: PullChosenChannelRequest, reader: jspb.BinaryReader): PullChosenChannelRequest;
}

export namespace PullChosenChannelRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullChosenChannelResponse extends jspb.Message {
  getChangesList(): Array<PullChosenChannelResponse.Change>;
  setChangesList(value: Array<PullChosenChannelResponse.Change>): PullChosenChannelResponse;
  clearChangesList(): PullChosenChannelResponse;
  addChanges(value?: PullChosenChannelResponse.Change, index?: number): PullChosenChannelResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullChosenChannelResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullChosenChannelResponse): PullChosenChannelResponse.AsObject;
  static serializeBinaryToWriter(message: PullChosenChannelResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullChosenChannelResponse;
  static deserializeBinaryFromReader(message: PullChosenChannelResponse, reader: jspb.BinaryReader): PullChosenChannelResponse;
}

export namespace PullChosenChannelResponse {
  export type AsObject = {
    changesList: Array<PullChosenChannelResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getChosenChannel(): Channel | undefined;
    setChosenChannel(value?: Channel): Change;
    hasChosenChannel(): boolean;
    clearChosenChannel(): Change;

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
      chosenChannel?: Channel.AsObject;
    };
  }

}

export class DescribeChosenChannelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeChosenChannelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeChosenChannelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeChosenChannelRequest): DescribeChosenChannelRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeChosenChannelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeChosenChannelRequest;
  static deserializeBinaryFromReader(message: DescribeChosenChannelRequest, reader: jspb.BinaryReader): DescribeChosenChannelRequest;
}

export namespace DescribeChosenChannelRequest {
  export type AsObject = {
    name: string;
  };
}

