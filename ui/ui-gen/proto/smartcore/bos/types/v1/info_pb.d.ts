import * as jspb from 'google-protobuf'

import * as google_protobuf_duration_pb from 'google-protobuf/google/protobuf/duration_pb'; // proto import: "google/protobuf/duration.proto"
import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"


export class ResourceSupport extends jspb.Message {
  getReadable(): boolean;
  setReadable(value: boolean): ResourceSupport;

  getWritable(): boolean;
  setWritable(value: boolean): ResourceSupport;

  getObservable(): boolean;
  setObservable(value: boolean): ResourceSupport;

  getWritableFields(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setWritableFields(value?: google_protobuf_field_mask_pb.FieldMask): ResourceSupport;
  hasWritableFields(): boolean;
  clearWritableFields(): ResourceSupport;

  getPullSupport(): PullSupport;
  setPullSupport(value: PullSupport): ResourceSupport;

  getPullPoll(): google_protobuf_duration_pb.Duration | undefined;
  setPullPoll(value?: google_protobuf_duration_pb.Duration): ResourceSupport;
  hasPullPoll(): boolean;
  clearPullPoll(): ResourceSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ResourceSupport.AsObject;
  static toObject(includeInstance: boolean, msg: ResourceSupport): ResourceSupport.AsObject;
  static serializeBinaryToWriter(message: ResourceSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ResourceSupport;
  static deserializeBinaryFromReader(message: ResourceSupport, reader: jspb.BinaryReader): ResourceSupport;
}

export namespace ResourceSupport {
  export type AsObject = {
    readable: boolean;
    writable: boolean;
    observable: boolean;
    writableFields?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pullSupport: PullSupport;
    pullPoll?: google_protobuf_duration_pb.Duration.AsObject;
  };
}

export enum PullSupport {
  PULL_SUPPORT_UNSPECIFIED = 0,
  PULL_SUPPORT_NATIVE = 1,
  PULL_SUPPORT_EMULATED = 2,
}
