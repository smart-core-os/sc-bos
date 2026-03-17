import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"


export class ModeValues extends jspb.Message {
  getValuesMap(): jspb.Map<string, string>;
  clearValuesMap(): ModeValues;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ModeValues.AsObject;
  static toObject(includeInstance: boolean, msg: ModeValues): ModeValues.AsObject;
  static serializeBinaryToWriter(message: ModeValues, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ModeValues;
  static deserializeBinaryFromReader(message: ModeValues, reader: jspb.BinaryReader): ModeValues;
}

export namespace ModeValues {
  export type AsObject = {
    valuesMap: Array<[string, string]>;
  };
}

export class ModeValuesRelative extends jspb.Message {
  getValuesMap(): jspb.Map<string, number>;
  clearValuesMap(): ModeValuesRelative;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ModeValuesRelative.AsObject;
  static toObject(includeInstance: boolean, msg: ModeValuesRelative): ModeValuesRelative.AsObject;
  static serializeBinaryToWriter(message: ModeValuesRelative, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ModeValuesRelative;
  static deserializeBinaryFromReader(message: ModeValuesRelative, reader: jspb.BinaryReader): ModeValuesRelative;
}

export namespace ModeValuesRelative {
  export type AsObject = {
    valuesMap: Array<[string, number]>;
  };
}

export class Modes extends jspb.Message {
  getModesList(): Array<Modes.Mode>;
  setModesList(value: Array<Modes.Mode>): Modes;
  clearModesList(): Modes;
  addModes(value?: Modes.Mode, index?: number): Modes.Mode;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Modes.AsObject;
  static toObject(includeInstance: boolean, msg: Modes): Modes.AsObject;
  static serializeBinaryToWriter(message: Modes, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Modes;
  static deserializeBinaryFromReader(message: Modes, reader: jspb.BinaryReader): Modes;
}

export namespace Modes {
  export type AsObject = {
    modesList: Array<Modes.Mode.AsObject>;
  };

  export class Value extends jspb.Message {
    getName(): string;
    setName(value: string): Value;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Value.AsObject;
    static toObject(includeInstance: boolean, msg: Value): Value.AsObject;
    static serializeBinaryToWriter(message: Value, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Value;
    static deserializeBinaryFromReader(message: Value, reader: jspb.BinaryReader): Value;
  }

  export namespace Value {
    export type AsObject = {
      name: string;
    };
  }


  export class Mode extends jspb.Message {
    getName(): string;
    setName(value: string): Mode;

    getValuesList(): Array<Modes.Value>;
    setValuesList(value: Array<Modes.Value>): Mode;
    clearValuesList(): Mode;
    addValues(value?: Modes.Value, index?: number): Modes.Value;

    getOrdered(): boolean;
    setOrdered(value: boolean): Mode;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Mode.AsObject;
    static toObject(includeInstance: boolean, msg: Mode): Mode.AsObject;
    static serializeBinaryToWriter(message: Mode, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Mode;
    static deserializeBinaryFromReader(message: Mode, reader: jspb.BinaryReader): Mode;
  }

  export namespace Mode {
    export type AsObject = {
      name: string;
      valuesList: Array<Modes.Value.AsObject>;
      ordered: boolean;
    };
  }

}

export class ModesSupport extends jspb.Message {
  getModeValuesSupport(): types_info_pb.ResourceSupport | undefined;
  setModeValuesSupport(value?: types_info_pb.ResourceSupport): ModesSupport;
  hasModeValuesSupport(): boolean;
  clearModeValuesSupport(): ModesSupport;

  getAvailableModes(): Modes | undefined;
  setAvailableModes(value?: Modes): ModesSupport;
  hasAvailableModes(): boolean;
  clearAvailableModes(): ModesSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ModesSupport.AsObject;
  static toObject(includeInstance: boolean, msg: ModesSupport): ModesSupport.AsObject;
  static serializeBinaryToWriter(message: ModesSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ModesSupport;
  static deserializeBinaryFromReader(message: ModesSupport, reader: jspb.BinaryReader): ModesSupport;
}

export namespace ModesSupport {
  export type AsObject = {
    modeValuesSupport?: types_info_pb.ResourceSupport.AsObject;
    availableModes?: Modes.AsObject;
  };
}

export class GetModeValuesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetModeValuesRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetModeValuesRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetModeValuesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetModeValuesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetModeValuesRequest): GetModeValuesRequest.AsObject;
  static serializeBinaryToWriter(message: GetModeValuesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetModeValuesRequest;
  static deserializeBinaryFromReader(message: GetModeValuesRequest, reader: jspb.BinaryReader): GetModeValuesRequest;
}

export namespace GetModeValuesRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateModeValuesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateModeValuesRequest;

  getModeValues(): ModeValues | undefined;
  setModeValues(value?: ModeValues): UpdateModeValuesRequest;
  hasModeValues(): boolean;
  clearModeValues(): UpdateModeValuesRequest;

  getRelative(): ModeValuesRelative | undefined;
  setRelative(value?: ModeValuesRelative): UpdateModeValuesRequest;
  hasRelative(): boolean;
  clearRelative(): UpdateModeValuesRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateModeValuesRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateModeValuesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateModeValuesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateModeValuesRequest): UpdateModeValuesRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateModeValuesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateModeValuesRequest;
  static deserializeBinaryFromReader(message: UpdateModeValuesRequest, reader: jspb.BinaryReader): UpdateModeValuesRequest;
}

export namespace UpdateModeValuesRequest {
  export type AsObject = {
    name: string;
    modeValues?: ModeValues.AsObject;
    relative?: ModeValuesRelative.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullModeValuesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullModeValuesRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullModeValuesRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullModeValuesRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullModeValuesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullModeValuesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullModeValuesRequest): PullModeValuesRequest.AsObject;
  static serializeBinaryToWriter(message: PullModeValuesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullModeValuesRequest;
  static deserializeBinaryFromReader(message: PullModeValuesRequest, reader: jspb.BinaryReader): PullModeValuesRequest;
}

export namespace PullModeValuesRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullModeValuesResponse extends jspb.Message {
  getChangesList(): Array<PullModeValuesResponse.Change>;
  setChangesList(value: Array<PullModeValuesResponse.Change>): PullModeValuesResponse;
  clearChangesList(): PullModeValuesResponse;
  addChanges(value?: PullModeValuesResponse.Change, index?: number): PullModeValuesResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullModeValuesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullModeValuesResponse): PullModeValuesResponse.AsObject;
  static serializeBinaryToWriter(message: PullModeValuesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullModeValuesResponse;
  static deserializeBinaryFromReader(message: PullModeValuesResponse, reader: jspb.BinaryReader): PullModeValuesResponse;
}

export namespace PullModeValuesResponse {
  export type AsObject = {
    changesList: Array<PullModeValuesResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getModeValues(): ModeValues | undefined;
    setModeValues(value?: ModeValues): Change;
    hasModeValues(): boolean;
    clearModeValues(): Change;

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
      modeValues?: ModeValues.AsObject;
    };
  }

}

export class DescribeModesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeModesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeModesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeModesRequest): DescribeModesRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeModesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeModesRequest;
  static deserializeBinaryFromReader(message: DescribeModesRequest, reader: jspb.BinaryReader): DescribeModesRequest;
}

export namespace DescribeModesRequest {
  export type AsObject = {
    name: string;
  };
}

