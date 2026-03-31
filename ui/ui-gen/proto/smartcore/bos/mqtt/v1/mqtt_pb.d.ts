import * as jspb from 'google-protobuf'



export class PullMessagesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullMessagesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullMessagesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullMessagesRequest): PullMessagesRequest.AsObject;
  static serializeBinaryToWriter(message: PullMessagesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullMessagesRequest;
  static deserializeBinaryFromReader(message: PullMessagesRequest, reader: jspb.BinaryReader): PullMessagesRequest;
}

export namespace PullMessagesRequest {
  export type AsObject = {
    name: string;
  };
}

export class PullMessagesResponse extends jspb.Message {
  getName(): string;
  setName(value: string): PullMessagesResponse;

  getTopic(): string;
  setTopic(value: string): PullMessagesResponse;

  getPayload(): string;
  setPayload(value: string): PullMessagesResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullMessagesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullMessagesResponse): PullMessagesResponse.AsObject;
  static serializeBinaryToWriter(message: PullMessagesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullMessagesResponse;
  static deserializeBinaryFromReader(message: PullMessagesResponse, reader: jspb.BinaryReader): PullMessagesResponse;
}

export namespace PullMessagesResponse {
  export type AsObject = {
    name: string;
    topic: string;
    payload: string;
  };
}

