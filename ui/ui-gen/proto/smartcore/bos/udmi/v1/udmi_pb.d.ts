import * as jspb from 'google-protobuf'



export class PullControlTopicsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullControlTopicsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullControlTopicsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullControlTopicsRequest): PullControlTopicsRequest.AsObject;
  static serializeBinaryToWriter(message: PullControlTopicsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullControlTopicsRequest;
  static deserializeBinaryFromReader(message: PullControlTopicsRequest, reader: jspb.BinaryReader): PullControlTopicsRequest;
}

export namespace PullControlTopicsRequest {
  export type AsObject = {
    name: string;
  };
}

export class PullControlTopicsResponse extends jspb.Message {
  getName(): string;
  setName(value: string): PullControlTopicsResponse;

  getTopicsList(): Array<string>;
  setTopicsList(value: Array<string>): PullControlTopicsResponse;
  clearTopicsList(): PullControlTopicsResponse;
  addTopics(value: string, index?: number): PullControlTopicsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullControlTopicsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullControlTopicsResponse): PullControlTopicsResponse.AsObject;
  static serializeBinaryToWriter(message: PullControlTopicsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullControlTopicsResponse;
  static deserializeBinaryFromReader(message: PullControlTopicsResponse, reader: jspb.BinaryReader): PullControlTopicsResponse;
}

export namespace PullControlTopicsResponse {
  export type AsObject = {
    name: string;
    topicsList: Array<string>;
  };
}

export class OnMessageRequest extends jspb.Message {
  getName(): string;
  setName(value: string): OnMessageRequest;

  getMessage(): MqttMessage | undefined;
  setMessage(value?: MqttMessage): OnMessageRequest;
  hasMessage(): boolean;
  clearMessage(): OnMessageRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): OnMessageRequest.AsObject;
  static toObject(includeInstance: boolean, msg: OnMessageRequest): OnMessageRequest.AsObject;
  static serializeBinaryToWriter(message: OnMessageRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): OnMessageRequest;
  static deserializeBinaryFromReader(message: OnMessageRequest, reader: jspb.BinaryReader): OnMessageRequest;
}

export namespace OnMessageRequest {
  export type AsObject = {
    name: string;
    message?: MqttMessage.AsObject;
  };
}

export class OnMessageResponse extends jspb.Message {
  getName(): string;
  setName(value: string): OnMessageResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): OnMessageResponse.AsObject;
  static toObject(includeInstance: boolean, msg: OnMessageResponse): OnMessageResponse.AsObject;
  static serializeBinaryToWriter(message: OnMessageResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): OnMessageResponse;
  static deserializeBinaryFromReader(message: OnMessageResponse, reader: jspb.BinaryReader): OnMessageResponse;
}

export namespace OnMessageResponse {
  export type AsObject = {
    name: string;
  };
}

export class PullExportMessagesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullExportMessagesRequest;

  getIncludeLast(): boolean;
  setIncludeLast(value: boolean): PullExportMessagesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullExportMessagesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullExportMessagesRequest): PullExportMessagesRequest.AsObject;
  static serializeBinaryToWriter(message: PullExportMessagesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullExportMessagesRequest;
  static deserializeBinaryFromReader(message: PullExportMessagesRequest, reader: jspb.BinaryReader): PullExportMessagesRequest;
}

export namespace PullExportMessagesRequest {
  export type AsObject = {
    name: string;
    includeLast: boolean;
  };
}

export class PullExportMessagesResponse extends jspb.Message {
  getName(): string;
  setName(value: string): PullExportMessagesResponse;

  getMessage(): MqttMessage | undefined;
  setMessage(value?: MqttMessage): PullExportMessagesResponse;
  hasMessage(): boolean;
  clearMessage(): PullExportMessagesResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullExportMessagesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullExportMessagesResponse): PullExportMessagesResponse.AsObject;
  static serializeBinaryToWriter(message: PullExportMessagesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullExportMessagesResponse;
  static deserializeBinaryFromReader(message: PullExportMessagesResponse, reader: jspb.BinaryReader): PullExportMessagesResponse;
}

export namespace PullExportMessagesResponse {
  export type AsObject = {
    name: string;
    message?: MqttMessage.AsObject;
  };
}

export class GetExportMessageRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetExportMessageRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetExportMessageRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetExportMessageRequest): GetExportMessageRequest.AsObject;
  static serializeBinaryToWriter(message: GetExportMessageRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetExportMessageRequest;
  static deserializeBinaryFromReader(message: GetExportMessageRequest, reader: jspb.BinaryReader): GetExportMessageRequest;
}

export namespace GetExportMessageRequest {
  export type AsObject = {
    name: string;
  };
}

export class MqttMessage extends jspb.Message {
  getTopic(): string;
  setTopic(value: string): MqttMessage;

  getPayload(): string;
  setPayload(value: string): MqttMessage;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MqttMessage.AsObject;
  static toObject(includeInstance: boolean, msg: MqttMessage): MqttMessage.AsObject;
  static serializeBinaryToWriter(message: MqttMessage, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MqttMessage;
  static deserializeBinaryFromReader(message: MqttMessage, reader: jspb.BinaryReader): MqttMessage;
}

export namespace MqttMessage {
  export type AsObject = {
    topic: string;
    payload: string;
  };
}

