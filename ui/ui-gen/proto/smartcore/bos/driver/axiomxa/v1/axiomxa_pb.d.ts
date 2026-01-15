import * as jspb from 'google-protobuf'

import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class SaveQRCredentialRequest extends jspb.Message {
  getName(): string;
  setName(value: string): SaveQRCredentialRequest;

  getCardNumber(): number;
  setCardNumber(value: number): SaveQRCredentialRequest;

  getFirstName(): string;
  setFirstName(value: string): SaveQRCredentialRequest;

  getLastName(): string;
  setLastName(value: string): SaveQRCredentialRequest;

  getActiveTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setActiveTime(value?: google_protobuf_timestamp_pb.Timestamp): SaveQRCredentialRequest;
  hasActiveTime(): boolean;
  clearActiveTime(): SaveQRCredentialRequest;

  getExpireTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setExpireTime(value?: google_protobuf_timestamp_pb.Timestamp): SaveQRCredentialRequest;
  hasExpireTime(): boolean;
  clearExpireTime(): SaveQRCredentialRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SaveQRCredentialRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SaveQRCredentialRequest): SaveQRCredentialRequest.AsObject;
  static serializeBinaryToWriter(message: SaveQRCredentialRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SaveQRCredentialRequest;
  static deserializeBinaryFromReader(message: SaveQRCredentialRequest, reader: jspb.BinaryReader): SaveQRCredentialRequest;
}

export namespace SaveQRCredentialRequest {
  export type AsObject = {
    name: string;
    cardNumber: number;
    firstName: string;
    lastName: string;
    activeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    expireTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class SaveQRCredentialResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SaveQRCredentialResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SaveQRCredentialResponse): SaveQRCredentialResponse.AsObject;
  static serializeBinaryToWriter(message: SaveQRCredentialResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SaveQRCredentialResponse;
  static deserializeBinaryFromReader(message: SaveQRCredentialResponse, reader: jspb.BinaryReader): SaveQRCredentialResponse;
}

export namespace SaveQRCredentialResponse {
  export type AsObject = {
  };
}

