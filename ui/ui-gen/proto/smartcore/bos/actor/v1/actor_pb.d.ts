import * as jspb from 'google-protobuf'

import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_image_pb from '@smart-core-os/sc-api-grpc-web/types/image_pb'; // proto import: "types/image.proto"


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

  getVehicleRegistration(): string;
  setVehicleRegistration(value: string): Actor;

  getCompany(): string;
  setCompany(value: string): Actor;

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
    vehicleRegistration: string;
    company: string;
    idsMap: Array<[string, string]>;
    moreMap: Array<[string, string]>;
  };
}

