import * as jspb from 'google-protobuf'

import * as types_info_pb from '@smart-core-os/sc-api-grpc-web/types/info_pb'; // proto import: "types/info.proto"


export class Ticket extends jspb.Message {
  getId(): string;
  setId(value: string): Ticket;

  getSummary(): string;
  setSummary(value: string): Ticket;

  getDescription(): string;
  setDescription(value: string): Ticket;

  getReporterName(): string;
  setReporterName(value: string): Ticket;

  getClassification(): Ticket.Classification | undefined;
  setClassification(value?: Ticket.Classification): Ticket;
  hasClassification(): boolean;
  clearClassification(): Ticket;

  getSeverity(): Ticket.Severity | undefined;
  setSeverity(value?: Ticket.Severity): Ticket;
  hasSeverity(): boolean;
  clearSeverity(): Ticket;

  getExternalUrl(): string;
  setExternalUrl(value: string): Ticket;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Ticket.AsObject;
  static toObject(includeInstance: boolean, msg: Ticket): Ticket.AsObject;
  static serializeBinaryToWriter(message: Ticket, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Ticket;
  static deserializeBinaryFromReader(message: Ticket, reader: jspb.BinaryReader): Ticket;
}

export namespace Ticket {
  export type AsObject = {
    id: string;
    summary: string;
    description: string;
    reporterName: string;
    classification?: Ticket.Classification.AsObject;
    severity?: Ticket.Severity.AsObject;
    externalUrl: string;
  };

  export class Classification extends jspb.Message {
    getId(): string;
    setId(value: string): Classification;

    getTitle(): string;
    setTitle(value: string): Classification;

    getDescription(): string;
    setDescription(value: string): Classification;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Classification.AsObject;
    static toObject(includeInstance: boolean, msg: Classification): Classification.AsObject;
    static serializeBinaryToWriter(message: Classification, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Classification;
    static deserializeBinaryFromReader(message: Classification, reader: jspb.BinaryReader): Classification;
  }

  export namespace Classification {
    export type AsObject = {
      id: string;
      title: string;
      description: string;
    };
  }


  export class Severity extends jspb.Message {
    getId(): string;
    setId(value: string): Severity;

    getTitle(): string;
    setTitle(value: string): Severity;

    getDescription(): string;
    setDescription(value: string): Severity;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Severity.AsObject;
    static toObject(includeInstance: boolean, msg: Severity): Severity.AsObject;
    static serializeBinaryToWriter(message: Severity, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Severity;
    static deserializeBinaryFromReader(message: Severity, reader: jspb.BinaryReader): Severity;
  }

  export namespace Severity {
    export type AsObject = {
      id: string;
      title: string;
      description: string;
    };
  }

}

export class CreateTicketRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CreateTicketRequest;

  getTicket(): Ticket | undefined;
  setTicket(value?: Ticket): CreateTicketRequest;
  hasTicket(): boolean;
  clearTicket(): CreateTicketRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateTicketRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateTicketRequest): CreateTicketRequest.AsObject;
  static serializeBinaryToWriter(message: CreateTicketRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateTicketRequest;
  static deserializeBinaryFromReader(message: CreateTicketRequest, reader: jspb.BinaryReader): CreateTicketRequest;
}

export namespace CreateTicketRequest {
  export type AsObject = {
    name: string;
    ticket?: Ticket.AsObject;
  };
}

export class UpdateTicketRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateTicketRequest;

  getTicket(): Ticket | undefined;
  setTicket(value?: Ticket): UpdateTicketRequest;
  hasTicket(): boolean;
  clearTicket(): UpdateTicketRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateTicketRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateTicketRequest): UpdateTicketRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateTicketRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateTicketRequest;
  static deserializeBinaryFromReader(message: UpdateTicketRequest, reader: jspb.BinaryReader): UpdateTicketRequest;
}

export namespace UpdateTicketRequest {
  export type AsObject = {
    name: string;
    ticket?: Ticket.AsObject;
  };
}

export class DescribeTicketRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeTicketRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeTicketRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeTicketRequest): DescribeTicketRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeTicketRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeTicketRequest;
  static deserializeBinaryFromReader(message: DescribeTicketRequest, reader: jspb.BinaryReader): DescribeTicketRequest;
}

export namespace DescribeTicketRequest {
  export type AsObject = {
    name: string;
  };
}

export class TicketSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): TicketSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): TicketSupport;

  getClassificationsList(): Array<Ticket.Classification>;
  setClassificationsList(value: Array<Ticket.Classification>): TicketSupport;
  clearClassificationsList(): TicketSupport;
  addClassifications(value?: Ticket.Classification, index?: number): Ticket.Classification;

  getSeveritiesList(): Array<Ticket.Severity>;
  setSeveritiesList(value: Array<Ticket.Severity>): TicketSupport;
  clearSeveritiesList(): TicketSupport;
  addSeverities(value?: Ticket.Severity, index?: number): Ticket.Severity;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TicketSupport.AsObject;
  static toObject(includeInstance: boolean, msg: TicketSupport): TicketSupport.AsObject;
  static serializeBinaryToWriter(message: TicketSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TicketSupport;
  static deserializeBinaryFromReader(message: TicketSupport, reader: jspb.BinaryReader): TicketSupport;
}

export namespace TicketSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    classificationsList: Array<Ticket.Classification.AsObject>;
    severitiesList: Array<Ticket.Severity.AsObject>;
  };
}

