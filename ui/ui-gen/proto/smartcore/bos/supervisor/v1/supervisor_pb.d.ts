import * as jspb from 'google-protobuf'

import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class UpdateStatus extends jspb.Message {
  getState(): UpdateStatus.State;
  setState(value: UpdateStatus.State): UpdateStatus;

  getVersion(): string;
  setVersion(value: string): UpdateStatus;

  getError(): string;
  setError(value: string): UpdateStatus;

  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): UpdateStatus;
  hasStartTime(): boolean;
  clearStartTime(): UpdateStatus;

  getFinishTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setFinishTime(value?: google_protobuf_timestamp_pb.Timestamp): UpdateStatus;
  hasFinishTime(): boolean;
  clearFinishTime(): UpdateStatus;

  getDeploymentId(): string;
  setDeploymentId(value: string): UpdateStatus;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateStatus.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateStatus): UpdateStatus.AsObject;
  static serializeBinaryToWriter(message: UpdateStatus, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateStatus;
  static deserializeBinaryFromReader(message: UpdateStatus, reader: jspb.BinaryReader): UpdateStatus;
}

export namespace UpdateStatus {
  export type AsObject = {
    state: UpdateStatus.State;
    version: string;
    error: string;
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    finishTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    deploymentId: string;
  };

  export enum State {
    STATE_UNSPECIFIED = 0,
    IDLE = 1,
    DOWNLOADING = 2,
    INSTALLING = 3,
    COMPLETED = 4,
    FAILED = 5,
  }
}

export class InstallUpdateRequest extends jspb.Message {
  getVersion(): string;
  setVersion(value: string): InstallUpdateRequest;

  getDownloadUrl(): string;
  setDownloadUrl(value: string): InstallUpdateRequest;

  getSha256(): string;
  setSha256(value: string): InstallUpdateRequest;

  getDeploymentId(): string;
  setDeploymentId(value: string): InstallUpdateRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): InstallUpdateRequest.AsObject;
  static toObject(includeInstance: boolean, msg: InstallUpdateRequest): InstallUpdateRequest.AsObject;
  static serializeBinaryToWriter(message: InstallUpdateRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): InstallUpdateRequest;
  static deserializeBinaryFromReader(message: InstallUpdateRequest, reader: jspb.BinaryReader): InstallUpdateRequest;
}

export namespace InstallUpdateRequest {
  export type AsObject = {
    version: string;
    downloadUrl: string;
    sha256: string;
    deploymentId: string;
  };
}

export class InstallUpdateResponse extends jspb.Message {
  getStatus(): UpdateStatus | undefined;
  setStatus(value?: UpdateStatus): InstallUpdateResponse;
  hasStatus(): boolean;
  clearStatus(): InstallUpdateResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): InstallUpdateResponse.AsObject;
  static toObject(includeInstance: boolean, msg: InstallUpdateResponse): InstallUpdateResponse.AsObject;
  static serializeBinaryToWriter(message: InstallUpdateResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): InstallUpdateResponse;
  static deserializeBinaryFromReader(message: InstallUpdateResponse, reader: jspb.BinaryReader): InstallUpdateResponse;
}

export namespace InstallUpdateResponse {
  export type AsObject = {
    status?: UpdateStatus.AsObject;
  };
}

export class GetUpdateStatusRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetUpdateStatusRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetUpdateStatusRequest): GetUpdateStatusRequest.AsObject;
  static serializeBinaryToWriter(message: GetUpdateStatusRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetUpdateStatusRequest;
  static deserializeBinaryFromReader(message: GetUpdateStatusRequest, reader: jspb.BinaryReader): GetUpdateStatusRequest;
}

export namespace GetUpdateStatusRequest {
  export type AsObject = {
  };
}

export class GetUpdateStatusResponse extends jspb.Message {
  getStatus(): UpdateStatus | undefined;
  setStatus(value?: UpdateStatus): GetUpdateStatusResponse;
  hasStatus(): boolean;
  clearStatus(): GetUpdateStatusResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetUpdateStatusResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetUpdateStatusResponse): GetUpdateStatusResponse.AsObject;
  static serializeBinaryToWriter(message: GetUpdateStatusResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetUpdateStatusResponse;
  static deserializeBinaryFromReader(message: GetUpdateStatusResponse, reader: jspb.BinaryReader): GetUpdateStatusResponse;
}

export namespace GetUpdateStatusResponse {
  export type AsObject = {
    status?: UpdateStatus.AsObject;
  };
}

export class CommitRequest extends jspb.Message {
  getVersion(): string;
  setVersion(value: string): CommitRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CommitRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CommitRequest): CommitRequest.AsObject;
  static serializeBinaryToWriter(message: CommitRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CommitRequest;
  static deserializeBinaryFromReader(message: CommitRequest, reader: jspb.BinaryReader): CommitRequest;
}

export namespace CommitRequest {
  export type AsObject = {
    version: string;
  };
}

export class CommitResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CommitResponse.AsObject;
  static toObject(includeInstance: boolean, msg: CommitResponse): CommitResponse.AsObject;
  static serializeBinaryToWriter(message: CommitResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CommitResponse;
  static deserializeBinaryFromReader(message: CommitResponse, reader: jspb.BinaryReader): CommitResponse;
}

export namespace CommitResponse {
  export type AsObject = {
  };
}

