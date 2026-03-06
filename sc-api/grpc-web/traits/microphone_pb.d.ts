import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"
import * as types_number_pb from '../types/number_pb'; // proto import: "types/number.proto"
import * as types_unit_pb from '../types/unit_pb'; // proto import: "types/unit.proto"


export class GainSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): GainSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): GainSupport;

  getGainAttributes(): types_number_pb.FloatAttributes | undefined;
  setGainAttributes(value?: types_number_pb.FloatAttributes): GainSupport;
  hasGainAttributes(): boolean;
  clearGainAttributes(): GainSupport;

  getMuteSupport(): types_unit_pb.MuteSupport;
  setMuteSupport(value: types_unit_pb.MuteSupport): GainSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GainSupport.AsObject;
  static toObject(includeInstance: boolean, msg: GainSupport): GainSupport.AsObject;
  static serializeBinaryToWriter(message: GainSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GainSupport;
  static deserializeBinaryFromReader(message: GainSupport, reader: jspb.BinaryReader): GainSupport;
}

export namespace GainSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    gainAttributes?: types_number_pb.FloatAttributes.AsObject;
    muteSupport: types_unit_pb.MuteSupport;
  };
}

export class GetMicrophoneGainRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetMicrophoneGainRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetMicrophoneGainRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetMicrophoneGainRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetMicrophoneGainRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetMicrophoneGainRequest): GetMicrophoneGainRequest.AsObject;
  static serializeBinaryToWriter(message: GetMicrophoneGainRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetMicrophoneGainRequest;
  static deserializeBinaryFromReader(message: GetMicrophoneGainRequest, reader: jspb.BinaryReader): GetMicrophoneGainRequest;
}

export namespace GetMicrophoneGainRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateMicrophoneGainRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateMicrophoneGainRequest;

  getGain(): types_unit_pb.AudioLevel | undefined;
  setGain(value?: types_unit_pb.AudioLevel): UpdateMicrophoneGainRequest;
  hasGain(): boolean;
  clearGain(): UpdateMicrophoneGainRequest;

  getDelta(): boolean;
  setDelta(value: boolean): UpdateMicrophoneGainRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateMicrophoneGainRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateMicrophoneGainRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateMicrophoneGainRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateMicrophoneGainRequest): UpdateMicrophoneGainRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateMicrophoneGainRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateMicrophoneGainRequest;
  static deserializeBinaryFromReader(message: UpdateMicrophoneGainRequest, reader: jspb.BinaryReader): UpdateMicrophoneGainRequest;
}

export namespace UpdateMicrophoneGainRequest {
  export type AsObject = {
    name: string;
    gain?: types_unit_pb.AudioLevel.AsObject;
    delta: boolean;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullMicrophoneGainRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullMicrophoneGainRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullMicrophoneGainRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullMicrophoneGainRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullMicrophoneGainRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullMicrophoneGainRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullMicrophoneGainRequest): PullMicrophoneGainRequest.AsObject;
  static serializeBinaryToWriter(message: PullMicrophoneGainRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullMicrophoneGainRequest;
  static deserializeBinaryFromReader(message: PullMicrophoneGainRequest, reader: jspb.BinaryReader): PullMicrophoneGainRequest;
}

export namespace PullMicrophoneGainRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullMicrophoneGainResponse extends jspb.Message {
  getChangesList(): Array<types_unit_pb.AudioLevelChange>;
  setChangesList(value: Array<types_unit_pb.AudioLevelChange>): PullMicrophoneGainResponse;
  clearChangesList(): PullMicrophoneGainResponse;
  addChanges(value?: types_unit_pb.AudioLevelChange, index?: number): types_unit_pb.AudioLevelChange;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullMicrophoneGainResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullMicrophoneGainResponse): PullMicrophoneGainResponse.AsObject;
  static serializeBinaryToWriter(message: PullMicrophoneGainResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullMicrophoneGainResponse;
  static deserializeBinaryFromReader(message: PullMicrophoneGainResponse, reader: jspb.BinaryReader): PullMicrophoneGainResponse;
}

export namespace PullMicrophoneGainResponse {
  export type AsObject = {
    changesList: Array<types_unit_pb.AudioLevelChange.AsObject>;
  };
}

export class DescribeGainRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeGainRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeGainRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeGainRequest): DescribeGainRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeGainRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeGainRequest;
  static deserializeBinaryFromReader(message: DescribeGainRequest, reader: jspb.BinaryReader): DescribeGainRequest;
}

export namespace DescribeGainRequest {
  export type AsObject = {
    name: string;
  };
}

