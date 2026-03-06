import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"


export class Input extends jspb.Message {
  getVideoInput(): string;
  setVideoInput(value: string): Input;

  getAudioInput(): string;
  setAudioInput(value: string): Input;

  getIndependentAv(): boolean;
  setIndependentAv(value: boolean): Input;

  getOutput(): string;
  setOutput(value: string): Input;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Input.AsObject;
  static toObject(includeInstance: boolean, msg: Input): Input.AsObject;
  static serializeBinaryToWriter(message: Input, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Input;
  static deserializeBinaryFromReader(message: Input, reader: jspb.BinaryReader): Input;
}

export namespace Input {
  export type AsObject = {
    videoInput: string;
    audioInput: string;
    independentAv: boolean;
    output: string;
  };
}

export class InputSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): InputSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): InputSupport;

  getInputsList(): Array<AvPort>;
  setInputsList(value: Array<AvPort>): InputSupport;
  clearInputsList(): InputSupport;
  addInputs(value?: AvPort, index?: number): AvPort;

  getSupportedFeature(): InputSupport.Feature;
  setSupportedFeature(value: InputSupport.Feature): InputSupport;

  getOutputsList(): Array<AvPort>;
  setOutputsList(value: Array<AvPort>): InputSupport;
  clearOutputsList(): InputSupport;
  addOutputs(value?: AvPort, index?: number): AvPort;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): InputSupport.AsObject;
  static toObject(includeInstance: boolean, msg: InputSupport): InputSupport.AsObject;
  static serializeBinaryToWriter(message: InputSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): InputSupport;
  static deserializeBinaryFromReader(message: InputSupport, reader: jspb.BinaryReader): InputSupport;
}

export namespace InputSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    inputsList: Array<AvPort.AsObject>;
    supportedFeature: InputSupport.Feature;
    outputsList: Array<AvPort.AsObject>;
  };

  export enum Feature {
    FEATURE_UNSPECIFIED = 0,
    AV = 1,
    AUDIO_ONLY = 2,
    VIDEO_ONLY = 3,
    INDEPENDENT = 4,
  }
}

export class AvPort extends jspb.Message {
  getName(): string;
  setName(value: string): AvPort;

  getTitle(): string;
  setTitle(value: string): AvPort;

  getDescription(): string;
  setDescription(value: string): AvPort;

  getSupportedFeature(): InputSupport.Feature;
  setSupportedFeature(value: InputSupport.Feature): AvPort;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AvPort.AsObject;
  static toObject(includeInstance: boolean, msg: AvPort): AvPort.AsObject;
  static serializeBinaryToWriter(message: AvPort, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AvPort;
  static deserializeBinaryFromReader(message: AvPort, reader: jspb.BinaryReader): AvPort;
}

export namespace AvPort {
  export type AsObject = {
    name: string;
    title: string;
    description: string;
    supportedFeature: InputSupport.Feature;
  };
}

export class UpdateInputRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateInputRequest;

  getInput(): Input | undefined;
  setInput(value?: Input): UpdateInputRequest;
  hasInput(): boolean;
  clearInput(): UpdateInputRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateInputRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateInputRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateInputRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateInputRequest): UpdateInputRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateInputRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateInputRequest;
  static deserializeBinaryFromReader(message: UpdateInputRequest, reader: jspb.BinaryReader): UpdateInputRequest;
}

export namespace UpdateInputRequest {
  export type AsObject = {
    name: string;
    input?: Input.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class GetInputRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetInputRequest;

  getOutput(): string;
  setOutput(value: string): GetInputRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetInputRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetInputRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetInputRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetInputRequest): GetInputRequest.AsObject;
  static serializeBinaryToWriter(message: GetInputRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetInputRequest;
  static deserializeBinaryFromReader(message: GetInputRequest, reader: jspb.BinaryReader): GetInputRequest;
}

export namespace GetInputRequest {
  export type AsObject = {
    name: string;
    output: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullInputRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullInputRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullInputRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullInputRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullInputRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullInputRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullInputRequest): PullInputRequest.AsObject;
  static serializeBinaryToWriter(message: PullInputRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullInputRequest;
  static deserializeBinaryFromReader(message: PullInputRequest, reader: jspb.BinaryReader): PullInputRequest;
}

export namespace PullInputRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullInputResponse extends jspb.Message {
  getChangesList(): Array<PullInputResponse.Change>;
  setChangesList(value: Array<PullInputResponse.Change>): PullInputResponse;
  clearChangesList(): PullInputResponse;
  addChanges(value?: PullInputResponse.Change, index?: number): PullInputResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullInputResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullInputResponse): PullInputResponse.AsObject;
  static serializeBinaryToWriter(message: PullInputResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullInputResponse;
  static deserializeBinaryFromReader(message: PullInputResponse, reader: jspb.BinaryReader): PullInputResponse;
}

export namespace PullInputResponse {
  export type AsObject = {
    changesList: Array<PullInputResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getInput(): Input | undefined;
    setInput(value?: Input): Change;
    hasInput(): boolean;
    clearInput(): Change;

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
      input?: Input.AsObject;
    };
  }

}

export class DescribeInputRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeInputRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeInputRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeInputRequest): DescribeInputRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeInputRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeInputRequest;
  static deserializeBinaryFromReader(message: DescribeInputRequest, reader: jspb.BinaryReader): DescribeInputRequest;
}

export namespace DescribeInputRequest {
  export type AsObject = {
    name: string;
  };
}

