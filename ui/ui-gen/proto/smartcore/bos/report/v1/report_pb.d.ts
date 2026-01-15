import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class ListReportsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListReportsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListReportsRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListReportsRequest;

  getPageSize(): number;
  setPageSize(value: number): ListReportsRequest;

  getPageToken(): string;
  setPageToken(value: string): ListReportsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListReportsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListReportsRequest): ListReportsRequest.AsObject;
  static serializeBinaryToWriter(message: ListReportsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListReportsRequest;
  static deserializeBinaryFromReader(message: ListReportsRequest, reader: jspb.BinaryReader): ListReportsRequest;
}

export namespace ListReportsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
  };
}

export class ListReportsResponse extends jspb.Message {
  getReportsList(): Array<Report>;
  setReportsList(value: Array<Report>): ListReportsResponse;
  clearReportsList(): ListReportsResponse;
  addReports(value?: Report, index?: number): Report;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListReportsResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListReportsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListReportsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListReportsResponse): ListReportsResponse.AsObject;
  static serializeBinaryToWriter(message: ListReportsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListReportsResponse;
  static deserializeBinaryFromReader(message: ListReportsResponse, reader: jspb.BinaryReader): ListReportsResponse;
}

export namespace ListReportsResponse {
  export type AsObject = {
    reportsList: Array<Report.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class Report extends jspb.Message {
  getId(): string;
  setId(value: string): Report;

  getTitle(): string;
  setTitle(value: string): Report;

  getDescription(): string;
  setDescription(value: string): Report;

  getCreateTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreateTime(value?: google_protobuf_timestamp_pb.Timestamp): Report;
  hasCreateTime(): boolean;
  clearCreateTime(): Report;

  getMediaType(): string;
  setMediaType(value: string): Report;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Report.AsObject;
  static toObject(includeInstance: boolean, msg: Report): Report.AsObject;
  static serializeBinaryToWriter(message: Report, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Report;
  static deserializeBinaryFromReader(message: Report, reader: jspb.BinaryReader): Report;
}

export namespace Report {
  export type AsObject = {
    id: string;
    title: string;
    description: string;
    createTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    mediaType: string;
  };
}

export class GetDownloadReportUrlRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetDownloadReportUrlRequest;

  getId(): string;
  setId(value: string): GetDownloadReportUrlRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDownloadReportUrlRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetDownloadReportUrlRequest): GetDownloadReportUrlRequest.AsObject;
  static serializeBinaryToWriter(message: GetDownloadReportUrlRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDownloadReportUrlRequest;
  static deserializeBinaryFromReader(message: GetDownloadReportUrlRequest, reader: jspb.BinaryReader): GetDownloadReportUrlRequest;
}

export namespace GetDownloadReportUrlRequest {
  export type AsObject = {
    name: string;
    id: string;
  };
}

export class DownloadReportUrl extends jspb.Message {
  getUrl(): string;
  setUrl(value: string): DownloadReportUrl;

  getFilename(): string;
  setFilename(value: string): DownloadReportUrl;

  getMediaType(): string;
  setMediaType(value: string): DownloadReportUrl;

  getExpireAfterTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setExpireAfterTime(value?: google_protobuf_timestamp_pb.Timestamp): DownloadReportUrl;
  hasExpireAfterTime(): boolean;
  clearExpireAfterTime(): DownloadReportUrl;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DownloadReportUrl.AsObject;
  static toObject(includeInstance: boolean, msg: DownloadReportUrl): DownloadReportUrl.AsObject;
  static serializeBinaryToWriter(message: DownloadReportUrl, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DownloadReportUrl;
  static deserializeBinaryFromReader(message: DownloadReportUrl, reader: jspb.BinaryReader): DownloadReportUrl;
}

export namespace DownloadReportUrl {
  export type AsObject = {
    url: string;
    filename: string;
    mediaType: string;
    expireAfterTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

