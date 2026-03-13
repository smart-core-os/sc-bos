import * as jspb from 'google-protobuf'

import * as google_protobuf_duration_pb from 'google-protobuf/google/protobuf/duration_pb'; // proto import: "google/protobuf/duration.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class LogMessage extends jspb.Message {
  getTimestamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTimestamp(value?: google_protobuf_timestamp_pb.Timestamp): LogMessage;
  hasTimestamp(): boolean;
  clearTimestamp(): LogMessage;

  getLevel(): LogLevel.Level;
  setLevel(value: LogLevel.Level): LogMessage;

  getLogger(): string;
  setLogger(value: string): LogMessage;

  getMessage(): string;
  setMessage(value: string): LogMessage;

  getFieldsMap(): jspb.Map<string, string>;
  clearFieldsMap(): LogMessage;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LogMessage.AsObject;
  static toObject(includeInstance: boolean, msg: LogMessage): LogMessage.AsObject;
  static serializeBinaryToWriter(message: LogMessage, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LogMessage;
  static deserializeBinaryFromReader(message: LogMessage, reader: jspb.BinaryReader): LogMessage;
}

export namespace LogMessage {
  export type AsObject = {
    timestamp?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    level: LogLevel.Level;
    logger: string;
    message: string;
    fieldsMap: Array<[string, string]>;
  };
}

export class LogLevel extends jspb.Message {
  getLevel(): LogLevel.Level;
  setLevel(value: LogLevel.Level): LogLevel;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LogLevel.AsObject;
  static toObject(includeInstance: boolean, msg: LogLevel): LogLevel.AsObject;
  static serializeBinaryToWriter(message: LogLevel, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LogLevel;
  static deserializeBinaryFromReader(message: LogLevel, reader: jspb.BinaryReader): LogLevel;
}

export namespace LogLevel {
  export type AsObject = {
    level: LogLevel.Level;
  };

  export enum Level {
    LEVEL_UNSPECIFIED = 0,
    DEBUG = 1,
    INFO = 2,
    WARN = 3,
    ERROR = 4,
    DPANIC = 5,
    PANIC = 6,
    FATAL = 7,
  }
}

export class LogMetadata extends jspb.Message {
  getTotalSizeBytes(): number;
  setTotalSizeBytes(value: number): LogMetadata;

  getFileCount(): number;
  setFileCount(value: number): LogMetadata;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LogMetadata.AsObject;
  static toObject(includeInstance: boolean, msg: LogMetadata): LogMetadata.AsObject;
  static serializeBinaryToWriter(message: LogMetadata, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LogMetadata;
  static deserializeBinaryFromReader(message: LogMetadata, reader: jspb.BinaryReader): LogMetadata;
}

export namespace LogMetadata {
  export type AsObject = {
    totalSizeBytes: number;
    fileCount: number;
  };
}

export class PullLogMessagesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullLogMessagesRequest;

  getInitialCount(): number;
  setInitialCount(value: number): PullLogMessagesRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullLogMessagesRequest;

  getMinLevel(): LogLevel.Level;
  setMinLevel(value: LogLevel.Level): PullLogMessagesRequest;

  getLoggerFilter(): string;
  setLoggerFilter(value: string): PullLogMessagesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullLogMessagesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullLogMessagesRequest): PullLogMessagesRequest.AsObject;
  static serializeBinaryToWriter(message: PullLogMessagesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullLogMessagesRequest;
  static deserializeBinaryFromReader(message: PullLogMessagesRequest, reader: jspb.BinaryReader): PullLogMessagesRequest;
}

export namespace PullLogMessagesRequest {
  export type AsObject = {
    name: string;
    initialCount: number;
    updatesOnly: boolean;
    minLevel: LogLevel.Level;
    loggerFilter: string;
  };
}

export class PullLogMessagesResponse extends jspb.Message {
  getMessagesList(): Array<LogMessage>;
  setMessagesList(value: Array<LogMessage>): PullLogMessagesResponse;
  clearMessagesList(): PullLogMessagesResponse;
  addMessages(value?: LogMessage, index?: number): LogMessage;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullLogMessagesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullLogMessagesResponse): PullLogMessagesResponse.AsObject;
  static serializeBinaryToWriter(message: PullLogMessagesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullLogMessagesResponse;
  static deserializeBinaryFromReader(message: PullLogMessagesResponse, reader: jspb.BinaryReader): PullLogMessagesResponse;
}

export namespace PullLogMessagesResponse {
  export type AsObject = {
    messagesList: Array<LogMessage.AsObject>;
  };
}

export class GetLogLevelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetLogLevelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetLogLevelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetLogLevelRequest): GetLogLevelRequest.AsObject;
  static serializeBinaryToWriter(message: GetLogLevelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetLogLevelRequest;
  static deserializeBinaryFromReader(message: GetLogLevelRequest, reader: jspb.BinaryReader): GetLogLevelRequest;
}

export namespace GetLogLevelRequest {
  export type AsObject = {
    name: string;
  };
}

export class PullLogLevelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullLogLevelRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullLogLevelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullLogLevelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullLogLevelRequest): PullLogLevelRequest.AsObject;
  static serializeBinaryToWriter(message: PullLogLevelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullLogLevelRequest;
  static deserializeBinaryFromReader(message: PullLogLevelRequest, reader: jspb.BinaryReader): PullLogLevelRequest;
}

export namespace PullLogLevelRequest {
  export type AsObject = {
    name: string;
    updatesOnly: boolean;
  };
}

export class PullLogLevelResponse extends jspb.Message {
  getChangesList(): Array<PullLogLevelResponse.Change>;
  setChangesList(value: Array<PullLogLevelResponse.Change>): PullLogLevelResponse;
  clearChangesList(): PullLogLevelResponse;
  addChanges(value?: PullLogLevelResponse.Change, index?: number): PullLogLevelResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullLogLevelResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullLogLevelResponse): PullLogLevelResponse.AsObject;
  static serializeBinaryToWriter(message: PullLogLevelResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullLogLevelResponse;
  static deserializeBinaryFromReader(message: PullLogLevelResponse, reader: jspb.BinaryReader): PullLogLevelResponse;
}

export namespace PullLogLevelResponse {
  export type AsObject = {
    changesList: Array<PullLogLevelResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getLogLevel(): LogLevel | undefined;
    setLogLevel(value?: LogLevel): Change;
    hasLogLevel(): boolean;
    clearLogLevel(): Change;

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
      logLevel?: LogLevel.AsObject;
    };
  }

}

export class UpdateLogLevelRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateLogLevelRequest;

  getLogLevel(): LogLevel | undefined;
  setLogLevel(value?: LogLevel): UpdateLogLevelRequest;
  hasLogLevel(): boolean;
  clearLogLevel(): UpdateLogLevelRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateLogLevelRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateLogLevelRequest): UpdateLogLevelRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateLogLevelRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateLogLevelRequest;
  static deserializeBinaryFromReader(message: UpdateLogLevelRequest, reader: jspb.BinaryReader): UpdateLogLevelRequest;
}

export namespace UpdateLogLevelRequest {
  export type AsObject = {
    name: string;
    logLevel?: LogLevel.AsObject;
  };
}

export class GetLogMetadataRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetLogMetadataRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetLogMetadataRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetLogMetadataRequest): GetLogMetadataRequest.AsObject;
  static serializeBinaryToWriter(message: GetLogMetadataRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetLogMetadataRequest;
  static deserializeBinaryFromReader(message: GetLogMetadataRequest, reader: jspb.BinaryReader): GetLogMetadataRequest;
}

export namespace GetLogMetadataRequest {
  export type AsObject = {
    name: string;
  };
}

export class PullLogMetadataRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullLogMetadataRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullLogMetadataRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullLogMetadataRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullLogMetadataRequest): PullLogMetadataRequest.AsObject;
  static serializeBinaryToWriter(message: PullLogMetadataRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullLogMetadataRequest;
  static deserializeBinaryFromReader(message: PullLogMetadataRequest, reader: jspb.BinaryReader): PullLogMetadataRequest;
}

export namespace PullLogMetadataRequest {
  export type AsObject = {
    name: string;
    updatesOnly: boolean;
  };
}

export class PullLogMetadataResponse extends jspb.Message {
  getChangesList(): Array<PullLogMetadataResponse.Change>;
  setChangesList(value: Array<PullLogMetadataResponse.Change>): PullLogMetadataResponse;
  clearChangesList(): PullLogMetadataResponse;
  addChanges(value?: PullLogMetadataResponse.Change, index?: number): PullLogMetadataResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullLogMetadataResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullLogMetadataResponse): PullLogMetadataResponse.AsObject;
  static serializeBinaryToWriter(message: PullLogMetadataResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullLogMetadataResponse;
  static deserializeBinaryFromReader(message: PullLogMetadataResponse, reader: jspb.BinaryReader): PullLogMetadataResponse;
}

export namespace PullLogMetadataResponse {
  export type AsObject = {
    changesList: Array<PullLogMetadataResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getLogMetadata(): LogMetadata | undefined;
    setLogMetadata(value?: LogMetadata): Change;
    hasLogMetadata(): boolean;
    clearLogMetadata(): Change;

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
      logMetadata?: LogMetadata.AsObject;
    };
  }

}

export class GetDownloadLogUrlRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetDownloadLogUrlRequest;

  getIncludeRotated(): boolean;
  setIncludeRotated(value: boolean): GetDownloadLogUrlRequest;

  getUrlTtl(): google_protobuf_duration_pb.Duration | undefined;
  setUrlTtl(value?: google_protobuf_duration_pb.Duration): GetDownloadLogUrlRequest;
  hasUrlTtl(): boolean;
  clearUrlTtl(): GetDownloadLogUrlRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDownloadLogUrlRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetDownloadLogUrlRequest): GetDownloadLogUrlRequest.AsObject;
  static serializeBinaryToWriter(message: GetDownloadLogUrlRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDownloadLogUrlRequest;
  static deserializeBinaryFromReader(message: GetDownloadLogUrlRequest, reader: jspb.BinaryReader): GetDownloadLogUrlRequest;
}

export namespace GetDownloadLogUrlRequest {
  export type AsObject = {
    name: string;
    includeRotated: boolean;
    urlTtl?: google_protobuf_duration_pb.Duration.AsObject;
  };
}

export class GetDownloadLogUrlResponse extends jspb.Message {
  getFilesList(): Array<GetDownloadLogUrlResponse.LogFile>;
  setFilesList(value: Array<GetDownloadLogUrlResponse.LogFile>): GetDownloadLogUrlResponse;
  clearFilesList(): GetDownloadLogUrlResponse;
  addFiles(value?: GetDownloadLogUrlResponse.LogFile, index?: number): GetDownloadLogUrlResponse.LogFile;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDownloadLogUrlResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetDownloadLogUrlResponse): GetDownloadLogUrlResponse.AsObject;
  static serializeBinaryToWriter(message: GetDownloadLogUrlResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDownloadLogUrlResponse;
  static deserializeBinaryFromReader(message: GetDownloadLogUrlResponse, reader: jspb.BinaryReader): GetDownloadLogUrlResponse;
}

export namespace GetDownloadLogUrlResponse {
  export type AsObject = {
    filesList: Array<GetDownloadLogUrlResponse.LogFile.AsObject>;
  };

  export class LogFile extends jspb.Message {
    getUrl(): string;
    setUrl(value: string): LogFile;

    getFilename(): string;
    setFilename(value: string): LogFile;

    getSizeBytes(): number;
    setSizeBytes(value: number): LogFile;

    getContentType(): string;
    setContentType(value: string): LogFile;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): LogFile.AsObject;
    static toObject(includeInstance: boolean, msg: LogFile): LogFile.AsObject;
    static serializeBinaryToWriter(message: LogFile, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): LogFile;
    static deserializeBinaryFromReader(message: LogFile, reader: jspb.BinaryReader): LogFile;
  }

  export namespace LogFile {
    export type AsObject = {
      url: string;
      filename: string;
      sizeBytes: number;
      contentType: string;
    };
  }

}

