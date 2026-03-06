import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"


export class MeterReading extends jspb.Message {
  getUsage(): number;
  setUsage(value: number): MeterReading;

  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): MeterReading;
  hasStartTime(): boolean;
  clearStartTime(): MeterReading;

  getEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setEndTime(value?: google_protobuf_timestamp_pb.Timestamp): MeterReading;
  hasEndTime(): boolean;
  clearEndTime(): MeterReading;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MeterReading.AsObject;
  static toObject(includeInstance: boolean, msg: MeterReading): MeterReading.AsObject;
  static serializeBinaryToWriter(message: MeterReading, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MeterReading;
  static deserializeBinaryFromReader(message: MeterReading, reader: jspb.BinaryReader): MeterReading;
}

export namespace MeterReading {
  export type AsObject = {
    usage: number;
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    endTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class MeterReadingSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): MeterReadingSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): MeterReadingSupport;

  getUnit(): string;
  setUnit(value: string): MeterReadingSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MeterReadingSupport.AsObject;
  static toObject(includeInstance: boolean, msg: MeterReadingSupport): MeterReadingSupport.AsObject;
  static serializeBinaryToWriter(message: MeterReadingSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MeterReadingSupport;
  static deserializeBinaryFromReader(message: MeterReadingSupport, reader: jspb.BinaryReader): MeterReadingSupport;
}

export namespace MeterReadingSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    unit: string;
  };
}

export class GetMeterReadingRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetMeterReadingRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetMeterReadingRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetMeterReadingRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetMeterReadingRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetMeterReadingRequest): GetMeterReadingRequest.AsObject;
  static serializeBinaryToWriter(message: GetMeterReadingRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetMeterReadingRequest;
  static deserializeBinaryFromReader(message: GetMeterReadingRequest, reader: jspb.BinaryReader): GetMeterReadingRequest;
}

export namespace GetMeterReadingRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullMeterReadingsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullMeterReadingsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullMeterReadingsRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullMeterReadingsRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullMeterReadingsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullMeterReadingsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullMeterReadingsRequest): PullMeterReadingsRequest.AsObject;
  static serializeBinaryToWriter(message: PullMeterReadingsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullMeterReadingsRequest;
  static deserializeBinaryFromReader(message: PullMeterReadingsRequest, reader: jspb.BinaryReader): PullMeterReadingsRequest;
}

export namespace PullMeterReadingsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullMeterReadingsResponse extends jspb.Message {
  getChangesList(): Array<PullMeterReadingsResponse.Change>;
  setChangesList(value: Array<PullMeterReadingsResponse.Change>): PullMeterReadingsResponse;
  clearChangesList(): PullMeterReadingsResponse;
  addChanges(value?: PullMeterReadingsResponse.Change, index?: number): PullMeterReadingsResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullMeterReadingsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullMeterReadingsResponse): PullMeterReadingsResponse.AsObject;
  static serializeBinaryToWriter(message: PullMeterReadingsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullMeterReadingsResponse;
  static deserializeBinaryFromReader(message: PullMeterReadingsResponse, reader: jspb.BinaryReader): PullMeterReadingsResponse;
}

export namespace PullMeterReadingsResponse {
  export type AsObject = {
    changesList: Array<PullMeterReadingsResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getMeterReading(): MeterReading | undefined;
    setMeterReading(value?: MeterReading): Change;
    hasMeterReading(): boolean;
    clearMeterReading(): Change;

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
      meterReading?: MeterReading.AsObject;
    };
  }

}

export class DescribeMeterReadingRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeMeterReadingRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeMeterReadingRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeMeterReadingRequest): DescribeMeterReadingRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeMeterReadingRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeMeterReadingRequest;
  static deserializeBinaryFromReader(message: DescribeMeterReadingRequest, reader: jspb.BinaryReader): DescribeMeterReadingRequest;
}

export namespace DescribeMeterReadingRequest {
  export type AsObject = {
    name: string;
  };
}

