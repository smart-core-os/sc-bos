import * as jspb from 'google-protobuf'



export class ForceTraitValuesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ForceTraitValuesRequest;

  getValuesList(): Array<TraitValue>;
  setValuesList(value: Array<TraitValue>): ForceTraitValuesRequest;
  clearValuesList(): ForceTraitValuesRequest;
  addValues(value?: TraitValue, index?: number): TraitValue;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ForceTraitValuesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ForceTraitValuesRequest): ForceTraitValuesRequest.AsObject;
  static serializeBinaryToWriter(message: ForceTraitValuesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ForceTraitValuesRequest;
  static deserializeBinaryFromReader(message: ForceTraitValuesRequest, reader: jspb.BinaryReader): ForceTraitValuesRequest;
}

export namespace ForceTraitValuesRequest {
  export type AsObject = {
    name: string;
    valuesList: Array<TraitValue.AsObject>;
  };
}

export class TraitValue extends jspb.Message {
  getTrait(): string;
  setTrait(value: string): TraitValue;

  getValueProtojson(): string;
  setValueProtojson(value: string): TraitValue;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TraitValue.AsObject;
  static toObject(includeInstance: boolean, msg: TraitValue): TraitValue.AsObject;
  static serializeBinaryToWriter(message: TraitValue, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TraitValue;
  static deserializeBinaryFromReader(message: TraitValue, reader: jspb.BinaryReader): TraitValue;
}

export namespace TraitValue {
  export type AsObject = {
    trait: string;
    valueProtojson: string;
  };
}

export class ForceTraitValuesResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ForceTraitValuesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ForceTraitValuesResponse): ForceTraitValuesResponse.AsObject;
  static serializeBinaryToWriter(message: ForceTraitValuesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ForceTraitValuesResponse;
  static deserializeBinaryFromReader(message: ForceTraitValuesResponse, reader: jspb.BinaryReader): ForceTraitValuesResponse;
}

export namespace ForceTraitValuesResponse {
  export type AsObject = {
  };
}

export class SetDeviceAutomationsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): SetDeviceAutomationsRequest;

  getAutomationsList(): Array<TraitAutomation>;
  setAutomationsList(value: Array<TraitAutomation>): SetDeviceAutomationsRequest;
  clearAutomationsList(): SetDeviceAutomationsRequest;
  addAutomations(value?: TraitAutomation, index?: number): TraitAutomation;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SetDeviceAutomationsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SetDeviceAutomationsRequest): SetDeviceAutomationsRequest.AsObject;
  static serializeBinaryToWriter(message: SetDeviceAutomationsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SetDeviceAutomationsRequest;
  static deserializeBinaryFromReader(message: SetDeviceAutomationsRequest, reader: jspb.BinaryReader): SetDeviceAutomationsRequest;
}

export namespace SetDeviceAutomationsRequest {
  export type AsObject = {
    name: string;
    automationsList: Array<TraitAutomation.AsObject>;
  };
}

export class TraitAutomation extends jspb.Message {
  getTrait(): string;
  setTrait(value: string): TraitAutomation;

  getId(): string;
  setId(value: string): TraitAutomation;

  getActive(): boolean;
  setActive(value: boolean): TraitAutomation;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TraitAutomation.AsObject;
  static toObject(includeInstance: boolean, msg: TraitAutomation): TraitAutomation.AsObject;
  static serializeBinaryToWriter(message: TraitAutomation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TraitAutomation;
  static deserializeBinaryFromReader(message: TraitAutomation, reader: jspb.BinaryReader): TraitAutomation;
}

export namespace TraitAutomation {
  export type AsObject = {
    trait: string;
    id: string;
    active: boolean;
  };
}

export class SetDeviceAutomationsResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SetDeviceAutomationsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SetDeviceAutomationsResponse): SetDeviceAutomationsResponse.AsObject;
  static serializeBinaryToWriter(message: SetDeviceAutomationsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SetDeviceAutomationsResponse;
  static deserializeBinaryFromReader(message: SetDeviceAutomationsResponse, reader: jspb.BinaryReader): SetDeviceAutomationsResponse;
}

export namespace SetDeviceAutomationsResponse {
  export type AsObject = {
  };
}

