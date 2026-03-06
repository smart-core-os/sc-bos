import * as jspb from 'google-protobuf'

import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_connection_pb from '../types/connection_pb'; // proto import: "types/connection.proto"


export class HealthState extends jspb.Message {
  getConnection(): ConnectionHealth | undefined;
  setConnection(value?: ConnectionHealth): HealthState;
  hasConnection(): boolean;
  clearConnection(): HealthState;

  getComm(): CommHealth | undefined;
  setComm(value?: CommHealth): HealthState;
  hasComm(): boolean;
  clearComm(): HealthState;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): HealthState.AsObject;
  static toObject(includeInstance: boolean, msg: HealthState): HealthState.AsObject;
  static serializeBinaryToWriter(message: HealthState, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): HealthState;
  static deserializeBinaryFromReader(message: HealthState, reader: jspb.BinaryReader): HealthState;
}

export namespace HealthState {
  export type AsObject = {
    connection?: ConnectionHealth.AsObject;
    comm?: CommHealth.AsObject;
  };
}

export class ConnectionHealth extends jspb.Message {
  getStatus(): types_connection_pb.Connectivity;
  setStatus(value: types_connection_pb.Connectivity): ConnectionHealth;

  getConnectTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setConnectTime(value?: google_protobuf_timestamp_pb.Timestamp): ConnectionHealth;
  hasConnectTime(): boolean;
  clearConnectTime(): ConnectionHealth;

  getDisconnectTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setDisconnectTime(value?: google_protobuf_timestamp_pb.Timestamp): ConnectionHealth;
  hasDisconnectTime(): boolean;
  clearDisconnectTime(): ConnectionHealth;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ConnectionHealth.AsObject;
  static toObject(includeInstance: boolean, msg: ConnectionHealth): ConnectionHealth.AsObject;
  static serializeBinaryToWriter(message: ConnectionHealth, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ConnectionHealth;
  static deserializeBinaryFromReader(message: ConnectionHealth, reader: jspb.BinaryReader): ConnectionHealth;
}

export namespace ConnectionHealth {
  export type AsObject = {
    status: types_connection_pb.Connectivity;
    connectTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    disconnectTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class CommHealth extends jspb.Message {
  getStatus(): types_connection_pb.CommStatus;
  setStatus(value: types_connection_pb.CommStatus): CommHealth;

  getFailureTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setFailureTime(value?: google_protobuf_timestamp_pb.Timestamp): CommHealth;
  hasFailureTime(): boolean;
  clearFailureTime(): CommHealth;

  getSuccessTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setSuccessTime(value?: google_protobuf_timestamp_pb.Timestamp): CommHealth;
  hasSuccessTime(): boolean;
  clearSuccessTime(): CommHealth;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CommHealth.AsObject;
  static toObject(includeInstance: boolean, msg: CommHealth): CommHealth.AsObject;
  static serializeBinaryToWriter(message: CommHealth, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CommHealth;
  static deserializeBinaryFromReader(message: CommHealth, reader: jspb.BinaryReader): CommHealth;
}

export namespace CommHealth {
  export type AsObject = {
    status: types_connection_pb.CommStatus;
    failureTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    successTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class GetHealthStateRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetHealthStateRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetHealthStateRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetHealthStateRequest): GetHealthStateRequest.AsObject;
  static serializeBinaryToWriter(message: GetHealthStateRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetHealthStateRequest;
  static deserializeBinaryFromReader(message: GetHealthStateRequest, reader: jspb.BinaryReader): GetHealthStateRequest;
}

export namespace GetHealthStateRequest {
  export type AsObject = {
    name: string;
  };
}

export class PullHealthStatesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullHealthStatesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullHealthStatesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullHealthStatesRequest): PullHealthStatesRequest.AsObject;
  static serializeBinaryToWriter(message: PullHealthStatesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullHealthStatesRequest;
  static deserializeBinaryFromReader(message: PullHealthStatesRequest, reader: jspb.BinaryReader): PullHealthStatesRequest;
}

export namespace PullHealthStatesRequest {
  export type AsObject = {
    name: string;
  };
}

export class PullHealthStatesResponse extends jspb.Message {
  getChangesList(): Array<HealthStateChange>;
  setChangesList(value: Array<HealthStateChange>): PullHealthStatesResponse;
  clearChangesList(): PullHealthStatesResponse;
  addChanges(value?: HealthStateChange, index?: number): HealthStateChange;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullHealthStatesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullHealthStatesResponse): PullHealthStatesResponse.AsObject;
  static serializeBinaryToWriter(message: PullHealthStatesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullHealthStatesResponse;
  static deserializeBinaryFromReader(message: PullHealthStatesResponse, reader: jspb.BinaryReader): PullHealthStatesResponse;
}

export namespace PullHealthStatesResponse {
  export type AsObject = {
    changesList: Array<HealthStateChange.AsObject>;
  };
}

export class HealthStateChange extends jspb.Message {
  getName(): string;
  setName(value: string): HealthStateChange;

  getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): HealthStateChange;
  hasChangeTime(): boolean;
  clearChangeTime(): HealthStateChange;

  getHealth(): HealthState | undefined;
  setHealth(value?: HealthState): HealthStateChange;
  hasHealth(): boolean;
  clearHealth(): HealthStateChange;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): HealthStateChange.AsObject;
  static toObject(includeInstance: boolean, msg: HealthStateChange): HealthStateChange.AsObject;
  static serializeBinaryToWriter(message: HealthStateChange, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): HealthStateChange;
  static deserializeBinaryFromReader(message: HealthStateChange, reader: jspb.BinaryReader): HealthStateChange;
}

export namespace HealthStateChange {
  export type AsObject = {
    name: string;
    changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    health?: HealthState.AsObject;
  };
}

