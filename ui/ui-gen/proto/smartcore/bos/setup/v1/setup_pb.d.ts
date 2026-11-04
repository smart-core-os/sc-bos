import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"


export class SetupState extends jspb.Message {
  getClaimed(): boolean;
  setClaimed(value: boolean): SetupState;

  getNodeName(): string;
  setNodeName(value: string): SetupState;

  getVersion(): string;
  setVersion(value: string): SetupState;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SetupState.AsObject;
  static toObject(includeInstance: boolean, msg: SetupState): SetupState.AsObject;
  static serializeBinaryToWriter(message: SetupState, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SetupState;
  static deserializeBinaryFromReader(message: SetupState, reader: jspb.BinaryReader): SetupState;
}

export namespace SetupState {
  export type AsObject = {
    claimed: boolean;
    nodeName: string;
    version: string;
  };
}

export class GetSetupStateRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetSetupStateRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetSetupStateRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetSetupStateRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetSetupStateRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetSetupStateRequest): GetSetupStateRequest.AsObject;
  static serializeBinaryToWriter(message: GetSetupStateRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetSetupStateRequest;
  static deserializeBinaryFromReader(message: GetSetupStateRequest, reader: jspb.BinaryReader): GetSetupStateRequest;
}

export namespace GetSetupStateRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class CompleteSetupRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CompleteSetupRequest;

  getSetupCode(): string;
  setSetupCode(value: string): CompleteSetupRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CompleteSetupRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CompleteSetupRequest): CompleteSetupRequest.AsObject;
  static serializeBinaryToWriter(message: CompleteSetupRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CompleteSetupRequest;
  static deserializeBinaryFromReader(message: CompleteSetupRequest, reader: jspb.BinaryReader): CompleteSetupRequest;
}

export namespace CompleteSetupRequest {
  export type AsObject = {
    name: string;
    setupCode: string;
  };
}

export class CompleteSetupResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CompleteSetupResponse.AsObject;
  static toObject(includeInstance: boolean, msg: CompleteSetupResponse): CompleteSetupResponse.AsObject;
  static serializeBinaryToWriter(message: CompleteSetupResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CompleteSetupResponse;
  static deserializeBinaryFromReader(message: CompleteSetupResponse, reader: jspb.BinaryReader): CompleteSetupResponse;
}

export namespace CompleteSetupResponse {
  export type AsObject = {
  };
}

