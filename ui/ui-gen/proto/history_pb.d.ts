import * as jspb from 'google-protobuf'

import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class HistoryRecord extends jspb.Message {
  getId(): string;
  setId(value: string): HistoryRecord;

  getSource(): string;
  setSource(value: string): HistoryRecord;

  getCreateTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreateTime(value?: google_protobuf_timestamp_pb.Timestamp): HistoryRecord;
  hasCreateTime(): boolean;
  clearCreateTime(): HistoryRecord;

  getPayload(): Uint8Array | string;
  getPayload_asU8(): Uint8Array;
  getPayload_asB64(): string;
  setPayload(value: Uint8Array | string): HistoryRecord;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): HistoryRecord.AsObject;
  static toObject(includeInstance: boolean, msg: HistoryRecord): HistoryRecord.AsObject;
  static serializeBinaryToWriter(message: HistoryRecord, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): HistoryRecord;
  static deserializeBinaryFromReader(message: HistoryRecord, reader: jspb.BinaryReader): HistoryRecord;
}

export namespace HistoryRecord {
  export type AsObject = {
    id: string;
    source: string;
    createTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    payload: Uint8Array | string;
  };

  export class Query extends jspb.Message {
    getSourceEqual(): string;
    setSourceEqual(value: string): Query;
    hasSourceEqual(): boolean;
    clearSourceEqual(): Query;

    getFromRecord(): HistoryRecord | undefined;
    setFromRecord(value?: HistoryRecord): Query;
    hasFromRecord(): boolean;
    clearFromRecord(): Query;

    getToRecord(): HistoryRecord | undefined;
    setToRecord(value?: HistoryRecord): Query;
    hasToRecord(): boolean;
    clearToRecord(): Query;

    getSourceCase(): Query.SourceCase;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Query.AsObject;
    static toObject(includeInstance: boolean, msg: Query): Query.AsObject;
    static serializeBinaryToWriter(message: Query, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Query;
    static deserializeBinaryFromReader(message: Query, reader: jspb.BinaryReader): Query;
  }

  export namespace Query {
    export type AsObject = {
      sourceEqual?: string;
      fromRecord?: HistoryRecord.AsObject;
      toRecord?: HistoryRecord.AsObject;
    };

    export enum SourceCase {
      SOURCE_NOT_SET = 0,
      SOURCE_EQUAL = 1,
    }
  }

}

export class CreateHistoryRecordRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CreateHistoryRecordRequest;

  getRecord(): HistoryRecord | undefined;
  setRecord(value?: HistoryRecord): CreateHistoryRecordRequest;
  hasRecord(): boolean;
  clearRecord(): CreateHistoryRecordRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateHistoryRecordRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateHistoryRecordRequest): CreateHistoryRecordRequest.AsObject;
  static serializeBinaryToWriter(message: CreateHistoryRecordRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateHistoryRecordRequest;
  static deserializeBinaryFromReader(message: CreateHistoryRecordRequest, reader: jspb.BinaryReader): CreateHistoryRecordRequest;
}

export namespace CreateHistoryRecordRequest {
  export type AsObject = {
    name: string;
    record?: HistoryRecord.AsObject;
  };
}

export class ListHistoryRecordsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListHistoryRecordsRequest;

  getPageSize(): number;
  setPageSize(value: number): ListHistoryRecordsRequest;

  getPageToken(): string;
  setPageToken(value: string): ListHistoryRecordsRequest;

  getOrderBy(): string;
  setOrderBy(value: string): ListHistoryRecordsRequest;

  getQuery(): HistoryRecord.Query | undefined;
  setQuery(value?: HistoryRecord.Query): ListHistoryRecordsRequest;
  hasQuery(): boolean;
  clearQuery(): ListHistoryRecordsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListHistoryRecordsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListHistoryRecordsRequest): ListHistoryRecordsRequest.AsObject;
  static serializeBinaryToWriter(message: ListHistoryRecordsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListHistoryRecordsRequest;
  static deserializeBinaryFromReader(message: ListHistoryRecordsRequest, reader: jspb.BinaryReader): ListHistoryRecordsRequest;
}

export namespace ListHistoryRecordsRequest {
  export type AsObject = {
    name: string;
    pageSize: number;
    pageToken: string;
    orderBy: string;
    query?: HistoryRecord.Query.AsObject;
  };
}

export class ListHistoryRecordsResponse extends jspb.Message {
  getRecordsList(): Array<HistoryRecord>;
  setRecordsList(value: Array<HistoryRecord>): ListHistoryRecordsResponse;
  clearRecordsList(): ListHistoryRecordsResponse;
  addRecords(value?: HistoryRecord, index?: number): HistoryRecord;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListHistoryRecordsResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListHistoryRecordsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListHistoryRecordsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListHistoryRecordsResponse): ListHistoryRecordsResponse.AsObject;
  static serializeBinaryToWriter(message: ListHistoryRecordsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListHistoryRecordsResponse;
  static deserializeBinaryFromReader(message: ListHistoryRecordsResponse, reader: jspb.BinaryReader): ListHistoryRecordsResponse;
}

export namespace ListHistoryRecordsResponse {
  export type AsObject = {
    recordsList: Array<HistoryRecord.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

