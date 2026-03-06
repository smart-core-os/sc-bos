import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_change_pb from '../types/change_pb'; // proto import: "types/change.proto"


export class Publication extends jspb.Message {
  getId(): string;
  setId(value: string): Publication;

  getVersion(): string;
  setVersion(value: string): Publication;

  getBody(): Uint8Array | string;
  getBody_asU8(): Uint8Array;
  getBody_asB64(): string;
  setBody(value: Uint8Array | string): Publication;

  getAudience(): Publication.Audience | undefined;
  setAudience(value?: Publication.Audience): Publication;
  hasAudience(): boolean;
  clearAudience(): Publication;

  getPublishTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setPublishTime(value?: google_protobuf_timestamp_pb.Timestamp): Publication;
  hasPublishTime(): boolean;
  clearPublishTime(): Publication;

  getMediaType(): string;
  setMediaType(value: string): Publication;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Publication.AsObject;
  static toObject(includeInstance: boolean, msg: Publication): Publication.AsObject;
  static serializeBinaryToWriter(message: Publication, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Publication;
  static deserializeBinaryFromReader(message: Publication, reader: jspb.BinaryReader): Publication;
}

export namespace Publication {
  export type AsObject = {
    id: string;
    version: string;
    body: Uint8Array | string;
    audience?: Publication.Audience.AsObject;
    publishTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    mediaType: string;
  };

  export class Audience extends jspb.Message {
    getName(): string;
    setName(value: string): Audience;

    getReceipt(): Publication.Audience.Receipt;
    setReceipt(value: Publication.Audience.Receipt): Audience;

    getReceiptRejectedReason(): string;
    setReceiptRejectedReason(value: string): Audience;

    getReceiptTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setReceiptTime(value?: google_protobuf_timestamp_pb.Timestamp): Audience;
    hasReceiptTime(): boolean;
    clearReceiptTime(): Audience;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Audience.AsObject;
    static toObject(includeInstance: boolean, msg: Audience): Audience.AsObject;
    static serializeBinaryToWriter(message: Audience, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Audience;
    static deserializeBinaryFromReader(message: Audience, reader: jspb.BinaryReader): Audience;
  }

  export namespace Audience {
    export type AsObject = {
      name: string;
      receipt: Publication.Audience.Receipt;
      receiptRejectedReason: string;
      receiptTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };

    export enum Receipt {
      RECEIPT_UNSPECIFIED = 0,
      NO_SIGNAL = 1,
      ACCEPTED = 2,
      REJECTED = 3,
    }
  }

}

export class CreatePublicationRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CreatePublicationRequest;

  getPublication(): Publication | undefined;
  setPublication(value?: Publication): CreatePublicationRequest;
  hasPublication(): boolean;
  clearPublication(): CreatePublicationRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreatePublicationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreatePublicationRequest): CreatePublicationRequest.AsObject;
  static serializeBinaryToWriter(message: CreatePublicationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreatePublicationRequest;
  static deserializeBinaryFromReader(message: CreatePublicationRequest, reader: jspb.BinaryReader): CreatePublicationRequest;
}

export namespace CreatePublicationRequest {
  export type AsObject = {
    name: string;
    publication?: Publication.AsObject;
  };
}

export class GetPublicationRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetPublicationRequest;

  getId(): string;
  setId(value: string): GetPublicationRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetPublicationRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetPublicationRequest;

  getVersion(): string;
  setVersion(value: string): GetPublicationRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetPublicationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetPublicationRequest): GetPublicationRequest.AsObject;
  static serializeBinaryToWriter(message: GetPublicationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetPublicationRequest;
  static deserializeBinaryFromReader(message: GetPublicationRequest, reader: jspb.BinaryReader): GetPublicationRequest;
}

export namespace GetPublicationRequest {
  export type AsObject = {
    name: string;
    id: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    version: string;
  };
}

export class UpdatePublicationRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdatePublicationRequest;

  getPublication(): Publication | undefined;
  setPublication(value?: Publication): UpdatePublicationRequest;
  hasPublication(): boolean;
  clearPublication(): UpdatePublicationRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdatePublicationRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdatePublicationRequest;

  getVersion(): string;
  setVersion(value: string): UpdatePublicationRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdatePublicationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdatePublicationRequest): UpdatePublicationRequest.AsObject;
  static serializeBinaryToWriter(message: UpdatePublicationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdatePublicationRequest;
  static deserializeBinaryFromReader(message: UpdatePublicationRequest, reader: jspb.BinaryReader): UpdatePublicationRequest;
}

export namespace UpdatePublicationRequest {
  export type AsObject = {
    name: string;
    publication?: Publication.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    version: string;
  };
}

export class DeletePublicationRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DeletePublicationRequest;

  getId(): string;
  setId(value: string): DeletePublicationRequest;

  getVersion(): string;
  setVersion(value: string): DeletePublicationRequest;

  getAllowMissing(): boolean;
  setAllowMissing(value: boolean): DeletePublicationRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeletePublicationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeletePublicationRequest): DeletePublicationRequest.AsObject;
  static serializeBinaryToWriter(message: DeletePublicationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeletePublicationRequest;
  static deserializeBinaryFromReader(message: DeletePublicationRequest, reader: jspb.BinaryReader): DeletePublicationRequest;
}

export namespace DeletePublicationRequest {
  export type AsObject = {
    name: string;
    id: string;
    version: string;
    allowMissing: boolean;
  };
}

export class PullPublicationRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullPublicationRequest;

  getId(): string;
  setId(value: string): PullPublicationRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullPublicationRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullPublicationRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullPublicationRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullPublicationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullPublicationRequest): PullPublicationRequest.AsObject;
  static serializeBinaryToWriter(message: PullPublicationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullPublicationRequest;
  static deserializeBinaryFromReader(message: PullPublicationRequest, reader: jspb.BinaryReader): PullPublicationRequest;
}

export namespace PullPublicationRequest {
  export type AsObject = {
    name: string;
    id: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullPublicationResponse extends jspb.Message {
  getChangesList(): Array<PullPublicationResponse.Change>;
  setChangesList(value: Array<PullPublicationResponse.Change>): PullPublicationResponse;
  clearChangesList(): PullPublicationResponse;
  addChanges(value?: PullPublicationResponse.Change, index?: number): PullPublicationResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullPublicationResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullPublicationResponse): PullPublicationResponse.AsObject;
  static serializeBinaryToWriter(message: PullPublicationResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullPublicationResponse;
  static deserializeBinaryFromReader(message: PullPublicationResponse, reader: jspb.BinaryReader): PullPublicationResponse;
}

export namespace PullPublicationResponse {
  export type AsObject = {
    changesList: Array<PullPublicationResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getPublication(): Publication | undefined;
    setPublication(value?: Publication): Change;
    hasPublication(): boolean;
    clearPublication(): Change;

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
      publication?: Publication.AsObject;
    };
  }

}

export class ListPublicationsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListPublicationsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListPublicationsRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListPublicationsRequest;

  getPageSize(): number;
  setPageSize(value: number): ListPublicationsRequest;

  getPageToken(): string;
  setPageToken(value: string): ListPublicationsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListPublicationsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListPublicationsRequest): ListPublicationsRequest.AsObject;
  static serializeBinaryToWriter(message: ListPublicationsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListPublicationsRequest;
  static deserializeBinaryFromReader(message: ListPublicationsRequest, reader: jspb.BinaryReader): ListPublicationsRequest;
}

export namespace ListPublicationsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
  };
}

export class ListPublicationsResponse extends jspb.Message {
  getPublicationsList(): Array<Publication>;
  setPublicationsList(value: Array<Publication>): ListPublicationsResponse;
  clearPublicationsList(): ListPublicationsResponse;
  addPublications(value?: Publication, index?: number): Publication;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListPublicationsResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListPublicationsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListPublicationsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListPublicationsResponse): ListPublicationsResponse.AsObject;
  static serializeBinaryToWriter(message: ListPublicationsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListPublicationsResponse;
  static deserializeBinaryFromReader(message: ListPublicationsResponse, reader: jspb.BinaryReader): ListPublicationsResponse;
}

export namespace ListPublicationsResponse {
  export type AsObject = {
    publicationsList: Array<Publication.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class PullPublicationsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullPublicationsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullPublicationsRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullPublicationsRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullPublicationsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullPublicationsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullPublicationsRequest): PullPublicationsRequest.AsObject;
  static serializeBinaryToWriter(message: PullPublicationsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullPublicationsRequest;
  static deserializeBinaryFromReader(message: PullPublicationsRequest, reader: jspb.BinaryReader): PullPublicationsRequest;
}

export namespace PullPublicationsRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullPublicationsResponse extends jspb.Message {
  getChangesList(): Array<PullPublicationsResponse.Change>;
  setChangesList(value: Array<PullPublicationsResponse.Change>): PullPublicationsResponse;
  clearChangesList(): PullPublicationsResponse;
  addChanges(value?: PullPublicationsResponse.Change, index?: number): PullPublicationsResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullPublicationsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullPublicationsResponse): PullPublicationsResponse.AsObject;
  static serializeBinaryToWriter(message: PullPublicationsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullPublicationsResponse;
  static deserializeBinaryFromReader(message: PullPublicationsResponse, reader: jspb.BinaryReader): PullPublicationsResponse;
}

export namespace PullPublicationsResponse {
  export type AsObject = {
    changesList: Array<PullPublicationsResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getType(): types_change_pb.ChangeType;
    setType(value: types_change_pb.ChangeType): Change;

    getNewValue(): Publication | undefined;
    setNewValue(value?: Publication): Change;
    hasNewValue(): boolean;
    clearNewValue(): Change;

    getOldValue(): Publication | undefined;
    setOldValue(value?: Publication): Change;
    hasOldValue(): boolean;
    clearOldValue(): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

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
      type: types_change_pb.ChangeType;
      newValue?: Publication.AsObject;
      oldValue?: Publication.AsObject;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };
  }

}

export class AcknowledgePublicationRequest extends jspb.Message {
  getName(): string;
  setName(value: string): AcknowledgePublicationRequest;

  getId(): string;
  setId(value: string): AcknowledgePublicationRequest;

  getVersion(): string;
  setVersion(value: string): AcknowledgePublicationRequest;

  getReceipt(): Publication.Audience.Receipt;
  setReceipt(value: Publication.Audience.Receipt): AcknowledgePublicationRequest;

  getReceiptRejectedReason(): string;
  setReceiptRejectedReason(value: string): AcknowledgePublicationRequest;

  getAllowAcknowledged(): boolean;
  setAllowAcknowledged(value: boolean): AcknowledgePublicationRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AcknowledgePublicationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: AcknowledgePublicationRequest): AcknowledgePublicationRequest.AsObject;
  static serializeBinaryToWriter(message: AcknowledgePublicationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AcknowledgePublicationRequest;
  static deserializeBinaryFromReader(message: AcknowledgePublicationRequest, reader: jspb.BinaryReader): AcknowledgePublicationRequest;
}

export namespace AcknowledgePublicationRequest {
  export type AsObject = {
    name: string;
    id: string;
    version: string;
    receipt: Publication.Audience.Receipt;
    receiptRejectedReason: string;
    allowAcknowledged: boolean;
  };
}

