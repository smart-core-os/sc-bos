import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class AnprEvent extends jspb.Message {
  getEventTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEventTime(value?: google_protobuf_timestamp_pb.Timestamp): AnprEvent;
  hasEventTime(): boolean;
  clearEventTime(): AnprEvent;

  getRegistrationPlate(): string;
  setRegistrationPlate(value: string): AnprEvent;

  getCountry(): string;
  setCountry(value: string): AnprEvent;

  getArea(): string;
  setArea(value: string): AnprEvent;

  getConfidence(): number;
  setConfidence(value: number): AnprEvent;
  hasConfidence(): boolean;
  clearConfidence(): AnprEvent;

  getPlateType(): string;
  setPlateType(value: string): AnprEvent;

  getYear(): string;
  setYear(value: string): AnprEvent;

  getVehicleInfo(): AnprEvent.VehicleInfo | undefined;
  setVehicleInfo(value?: AnprEvent.VehicleInfo): AnprEvent;
  hasVehicleInfo(): boolean;
  clearVehicleInfo(): AnprEvent;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AnprEvent.AsObject;
  static toObject(includeInstance: boolean, msg: AnprEvent): AnprEvent.AsObject;
  static serializeBinaryToWriter(message: AnprEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AnprEvent;
  static deserializeBinaryFromReader(message: AnprEvent, reader: jspb.BinaryReader): AnprEvent;
}

export namespace AnprEvent {
  export type AsObject = {
    eventTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    registrationPlate: string;
    country: string;
    area: string;
    confidence?: number;
    plateType: string;
    year: string;
    vehicleInfo?: AnprEvent.VehicleInfo.AsObject;
  };

  export class VehicleInfo extends jspb.Message {
    getVehicleType(): string;
    setVehicleType(value: string): VehicleInfo;

    getColour(): string;
    setColour(value: string): VehicleInfo;

    getMake(): string;
    setMake(value: string): VehicleInfo;

    getModel(): string;
    setModel(value: string): VehicleInfo;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): VehicleInfo.AsObject;
    static toObject(includeInstance: boolean, msg: VehicleInfo): VehicleInfo.AsObject;
    static serializeBinaryToWriter(message: VehicleInfo, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): VehicleInfo;
    static deserializeBinaryFromReader(message: VehicleInfo, reader: jspb.BinaryReader): VehicleInfo;
  }

  export namespace VehicleInfo {
    export type AsObject = {
      vehicleType: string;
      colour: string;
      make: string;
      model: string;
    };
  }


  export enum ConfidenceCase {
    _CONFIDENCE_NOT_SET = 0,
    CONFIDENCE = 5,
  }
}

export class ListAnprEventsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListAnprEventsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListAnprEventsRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListAnprEventsRequest;

  getPageSize(): number;
  setPageSize(value: number): ListAnprEventsRequest;

  getPageToken(): string;
  setPageToken(value: string): ListAnprEventsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAnprEventsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListAnprEventsRequest): ListAnprEventsRequest.AsObject;
  static serializeBinaryToWriter(message: ListAnprEventsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAnprEventsRequest;
  static deserializeBinaryFromReader(message: ListAnprEventsRequest, reader: jspb.BinaryReader): ListAnprEventsRequest;
}

export namespace ListAnprEventsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
  };
}

export class ListAnprEventsResponse extends jspb.Message {
  getAnprEventsList(): Array<AnprEvent>;
  setAnprEventsList(value: Array<AnprEvent>): ListAnprEventsResponse;
  clearAnprEventsList(): ListAnprEventsResponse;
  addAnprEvents(value?: AnprEvent, index?: number): AnprEvent;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListAnprEventsResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListAnprEventsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListAnprEventsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListAnprEventsResponse): ListAnprEventsResponse.AsObject;
  static serializeBinaryToWriter(message: ListAnprEventsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListAnprEventsResponse;
  static deserializeBinaryFromReader(message: ListAnprEventsResponse, reader: jspb.BinaryReader): ListAnprEventsResponse;
}

export namespace ListAnprEventsResponse {
  export type AsObject = {
    anprEventsList: Array<AnprEvent.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class PullAnprEventsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullAnprEventsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullAnprEventsRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullAnprEventsRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullAnprEventsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullAnprEventsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullAnprEventsRequest): PullAnprEventsRequest.AsObject;
  static serializeBinaryToWriter(message: PullAnprEventsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullAnprEventsRequest;
  static deserializeBinaryFromReader(message: PullAnprEventsRequest, reader: jspb.BinaryReader): PullAnprEventsRequest;
}

export namespace PullAnprEventsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullAnprEventsResponse extends jspb.Message {
  getChangesList(): Array<PullAnprEventsResponse.Change>;
  setChangesList(value: Array<PullAnprEventsResponse.Change>): PullAnprEventsResponse;
  clearChangesList(): PullAnprEventsResponse;
  addChanges(value?: PullAnprEventsResponse.Change, index?: number): PullAnprEventsResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullAnprEventsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullAnprEventsResponse): PullAnprEventsResponse.AsObject;
  static serializeBinaryToWriter(message: PullAnprEventsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullAnprEventsResponse;
  static deserializeBinaryFromReader(message: PullAnprEventsResponse, reader: jspb.BinaryReader): PullAnprEventsResponse;
}

export namespace PullAnprEventsResponse {
  export type AsObject = {
    changesList: Array<PullAnprEventsResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getAnprEvent(): AnprEvent | undefined;
    setAnprEvent(value?: AnprEvent): Change;
    hasAnprEvent(): boolean;
    clearAnprEvent(): Change;

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
      anprEvent?: AnprEvent.AsObject;
    };
  }

}

