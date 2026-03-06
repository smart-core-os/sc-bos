import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_image_pb from '../types/image_pb'; // proto import: "types/image.proto"


export class AccessAttempt extends jspb.Message {
  getGrant(): AccessAttempt.Grant;
  setGrant(value: AccessAttempt.Grant): AccessAttempt;

  getReason(): string;
  setReason(value: string): AccessAttempt;

  getActor(): AccessAttempt.Actor | undefined;
  setActor(value?: AccessAttempt.Actor): AccessAttempt;
  hasActor(): boolean;
  clearActor(): AccessAttempt;

  getAccessAttemptTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setAccessAttemptTime(value?: google_protobuf_timestamp_pb.Timestamp): AccessAttempt;
  hasAccessAttemptTime(): boolean;
  clearAccessAttemptTime(): AccessAttempt;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AccessAttempt.AsObject;
  static toObject(includeInstance: boolean, msg: AccessAttempt): AccessAttempt.AsObject;
  static serializeBinaryToWriter(message: AccessAttempt, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AccessAttempt;
  static deserializeBinaryFromReader(message: AccessAttempt, reader: jspb.BinaryReader): AccessAttempt;
}

export namespace AccessAttempt {
  export type AsObject = {
    grant: AccessAttempt.Grant;
    reason: string;
    actor?: AccessAttempt.Actor.AsObject;
    accessAttemptTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };

  export class Actor extends jspb.Message {
    getName(): string;
    setName(value: string): Actor;

    getTitle(): string;
    setTitle(value: string): Actor;

    getDisplayName(): string;
    setDisplayName(value: string): Actor;

    getPicture(): types_image_pb.Image | undefined;
    setPicture(value?: types_image_pb.Image): Actor;
    hasPicture(): boolean;
    clearPicture(): Actor;

    getUrl(): string;
    setUrl(value: string): Actor;

    getEmail(): string;
    setEmail(value: string): Actor;

    getLastGrantTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setLastGrantTime(value?: google_protobuf_timestamp_pb.Timestamp): Actor;
    hasLastGrantTime(): boolean;
    clearLastGrantTime(): Actor;

    getLastGrantZone(): string;
    setLastGrantZone(value: string): Actor;

    getIdsMap(): jspb.Map<string, string>;
    clearIdsMap(): Actor;

    getMoreMap(): jspb.Map<string, string>;
    clearMoreMap(): Actor;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Actor.AsObject;
    static toObject(includeInstance: boolean, msg: Actor): Actor.AsObject;
    static serializeBinaryToWriter(message: Actor, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Actor;
    static deserializeBinaryFromReader(message: Actor, reader: jspb.BinaryReader): Actor;
  }

  export namespace Actor {
    export type AsObject = {
      name: string;
      title: string;
      displayName: string;
      picture?: types_image_pb.Image.AsObject;
      url: string;
      email: string;
      lastGrantTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      lastGrantZone: string;
      idsMap: Array<[string, string]>;
      moreMap: Array<[string, string]>;
    };
  }


  export enum Grant {
    GRANT_UNKNOWN = 0,
    GRANTED = 1,
    DENIED = 2,
    PENDING = 3,
    ABORTED = 4,
    FORCED = 5,
    FAILED = 6,
    TAILGATE = 7,
  }
}

export class GetLastAccessAttemptRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetLastAccessAttemptRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetLastAccessAttemptRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetLastAccessAttemptRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetLastAccessAttemptRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetLastAccessAttemptRequest): GetLastAccessAttemptRequest.AsObject;
  static serializeBinaryToWriter(message: GetLastAccessAttemptRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetLastAccessAttemptRequest;
  static deserializeBinaryFromReader(message: GetLastAccessAttemptRequest, reader: jspb.BinaryReader): GetLastAccessAttemptRequest;
}

export namespace GetLastAccessAttemptRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullAccessAttemptsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullAccessAttemptsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullAccessAttemptsRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullAccessAttemptsRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullAccessAttemptsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullAccessAttemptsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullAccessAttemptsRequest): PullAccessAttemptsRequest.AsObject;
  static serializeBinaryToWriter(message: PullAccessAttemptsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullAccessAttemptsRequest;
  static deserializeBinaryFromReader(message: PullAccessAttemptsRequest, reader: jspb.BinaryReader): PullAccessAttemptsRequest;
}

export namespace PullAccessAttemptsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullAccessAttemptsResponse extends jspb.Message {
  getChangesList(): Array<PullAccessAttemptsResponse.Change>;
  setChangesList(value: Array<PullAccessAttemptsResponse.Change>): PullAccessAttemptsResponse;
  clearChangesList(): PullAccessAttemptsResponse;
  addChanges(value?: PullAccessAttemptsResponse.Change, index?: number): PullAccessAttemptsResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullAccessAttemptsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullAccessAttemptsResponse): PullAccessAttemptsResponse.AsObject;
  static serializeBinaryToWriter(message: PullAccessAttemptsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullAccessAttemptsResponse;
  static deserializeBinaryFromReader(message: PullAccessAttemptsResponse, reader: jspb.BinaryReader): PullAccessAttemptsResponse;
}

export namespace PullAccessAttemptsResponse {
  export type AsObject = {
    changesList: Array<PullAccessAttemptsResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getAccessAttempt(): AccessAttempt | undefined;
    setAccessAttempt(value?: AccessAttempt): Change;
    hasAccessAttempt(): boolean;
    clearAccessAttempt(): Change;

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
      accessAttempt?: AccessAttempt.AsObject;
    };
  }

}

