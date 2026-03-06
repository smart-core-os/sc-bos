import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"
import * as types_number_pb from '../types/number_pb'; // proto import: "types/number.proto"
import * as types_unit_pb from '../types/unit_pb'; // proto import: "types/unit.proto"


export class VolumeSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): VolumeSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): VolumeSupport;

  getVolumeAttributes(): types_number_pb.FloatAttributes | undefined;
  setVolumeAttributes(value?: types_number_pb.FloatAttributes): VolumeSupport;
  hasVolumeAttributes(): boolean;
  clearVolumeAttributes(): VolumeSupport;

  getMuteSupport(): types_unit_pb.MuteSupport;
  setMuteSupport(value: types_unit_pb.MuteSupport): VolumeSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): VolumeSupport.AsObject;
  static toObject(includeInstance: boolean, msg: VolumeSupport): VolumeSupport.AsObject;
  static serializeBinaryToWriter(message: VolumeSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): VolumeSupport;
  static deserializeBinaryFromReader(message: VolumeSupport, reader: jspb.BinaryReader): VolumeSupport;
}

export namespace VolumeSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    volumeAttributes?: types_number_pb.FloatAttributes.AsObject;
    muteSupport: types_unit_pb.MuteSupport;
  };
}

export class GetSpeakerVolumeRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetSpeakerVolumeRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetSpeakerVolumeRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetSpeakerVolumeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetSpeakerVolumeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetSpeakerVolumeRequest): GetSpeakerVolumeRequest.AsObject;
  static serializeBinaryToWriter(message: GetSpeakerVolumeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetSpeakerVolumeRequest;
  static deserializeBinaryFromReader(message: GetSpeakerVolumeRequest, reader: jspb.BinaryReader): GetSpeakerVolumeRequest;
}

export namespace GetSpeakerVolumeRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateSpeakerVolumeRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateSpeakerVolumeRequest;

  getVolume(): types_unit_pb.AudioLevel | undefined;
  setVolume(value?: types_unit_pb.AudioLevel): UpdateSpeakerVolumeRequest;
  hasVolume(): boolean;
  clearVolume(): UpdateSpeakerVolumeRequest;

  getDelta(): boolean;
  setDelta(value: boolean): UpdateSpeakerVolumeRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateSpeakerVolumeRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateSpeakerVolumeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateSpeakerVolumeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateSpeakerVolumeRequest): UpdateSpeakerVolumeRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateSpeakerVolumeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateSpeakerVolumeRequest;
  static deserializeBinaryFromReader(message: UpdateSpeakerVolumeRequest, reader: jspb.BinaryReader): UpdateSpeakerVolumeRequest;
}

export namespace UpdateSpeakerVolumeRequest {
  export type AsObject = {
    name: string;
    volume?: types_unit_pb.AudioLevel.AsObject;
    delta: boolean;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullSpeakerVolumeRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullSpeakerVolumeRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullSpeakerVolumeRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullSpeakerVolumeRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullSpeakerVolumeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullSpeakerVolumeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullSpeakerVolumeRequest): PullSpeakerVolumeRequest.AsObject;
  static serializeBinaryToWriter(message: PullSpeakerVolumeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullSpeakerVolumeRequest;
  static deserializeBinaryFromReader(message: PullSpeakerVolumeRequest, reader: jspb.BinaryReader): PullSpeakerVolumeRequest;
}

export namespace PullSpeakerVolumeRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullSpeakerVolumeResponse extends jspb.Message {
  getChangesList(): Array<types_unit_pb.AudioLevelChange>;
  setChangesList(value: Array<types_unit_pb.AudioLevelChange>): PullSpeakerVolumeResponse;
  clearChangesList(): PullSpeakerVolumeResponse;
  addChanges(value?: types_unit_pb.AudioLevelChange, index?: number): types_unit_pb.AudioLevelChange;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullSpeakerVolumeResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullSpeakerVolumeResponse): PullSpeakerVolumeResponse.AsObject;
  static serializeBinaryToWriter(message: PullSpeakerVolumeResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullSpeakerVolumeResponse;
  static deserializeBinaryFromReader(message: PullSpeakerVolumeResponse, reader: jspb.BinaryReader): PullSpeakerVolumeResponse;
}

export namespace PullSpeakerVolumeResponse {
  export type AsObject = {
    changesList: Array<types_unit_pb.AudioLevelChange.AsObject>;
  };
}

export class DescribeVolumeRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeVolumeRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeVolumeRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeVolumeRequest): DescribeVolumeRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeVolumeRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeVolumeRequest;
  static deserializeBinaryFromReader(message: DescribeVolumeRequest, reader: jspb.BinaryReader): DescribeVolumeRequest;
}

export namespace DescribeVolumeRequest {
  export type AsObject = {
    name: string;
  };
}

