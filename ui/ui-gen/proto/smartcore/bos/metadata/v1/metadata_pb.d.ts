import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class Metadata extends jspb.Message {
  getName(): string;
  setName(value: string): Metadata;

  getTraitsList(): Array<TraitMetadata>;
  setTraitsList(value: Array<TraitMetadata>): Metadata;
  clearTraitsList(): Metadata;
  addTraits(value?: TraitMetadata, index?: number): TraitMetadata;

  getAppearance(): Metadata.Appearance | undefined;
  setAppearance(value?: Metadata.Appearance): Metadata;
  hasAppearance(): boolean;
  clearAppearance(): Metadata;

  getLocation(): Metadata.Location | undefined;
  setLocation(value?: Metadata.Location): Metadata;
  hasLocation(): boolean;
  clearLocation(): Metadata;

  getId(): Metadata.ID | undefined;
  setId(value?: Metadata.ID): Metadata;
  hasId(): boolean;
  clearId(): Metadata;

  getProduct(): Metadata.Product | undefined;
  setProduct(value?: Metadata.Product): Metadata;
  hasProduct(): boolean;
  clearProduct(): Metadata;

  getRevision(): Metadata.Revision | undefined;
  setRevision(value?: Metadata.Revision): Metadata;
  hasRevision(): boolean;
  clearRevision(): Metadata;

  getInstallation(): Metadata.Installation | undefined;
  setInstallation(value?: Metadata.Installation): Metadata;
  hasInstallation(): boolean;
  clearInstallation(): Metadata;

  getNicsList(): Array<Metadata.NIC>;
  setNicsList(value: Array<Metadata.NIC>): Metadata;
  clearNicsList(): Metadata;
  addNics(value?: Metadata.NIC, index?: number): Metadata.NIC;

  getMembership(): Metadata.Membership | undefined;
  setMembership(value?: Metadata.Membership): Metadata;
  hasMembership(): boolean;
  clearMembership(): Metadata;

  getMoreMap(): jspb.Map<string, string>;
  clearMoreMap(): Metadata;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Metadata.AsObject;
  static toObject(includeInstance: boolean, msg: Metadata): Metadata.AsObject;
  static serializeBinaryToWriter(message: Metadata, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Metadata;
  static deserializeBinaryFromReader(message: Metadata, reader: jspb.BinaryReader): Metadata;
}

export namespace Metadata {
  export type AsObject = {
    name: string;
    traitsList: Array<TraitMetadata.AsObject>;
    appearance?: Metadata.Appearance.AsObject;
    location?: Metadata.Location.AsObject;
    id?: Metadata.ID.AsObject;
    product?: Metadata.Product.AsObject;
    revision?: Metadata.Revision.AsObject;
    installation?: Metadata.Installation.AsObject;
    nicsList: Array<Metadata.NIC.AsObject>;
    membership?: Metadata.Membership.AsObject;
    moreMap: Array<[string, string]>;
  };

  export class Appearance extends jspb.Message {
    getTitle(): string;
    setTitle(value: string): Appearance;

    getDescription(): string;
    setDescription(value: string): Appearance;

    getMoreMap(): jspb.Map<string, string>;
    clearMoreMap(): Appearance;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Appearance.AsObject;
    static toObject(includeInstance: boolean, msg: Appearance): Appearance.AsObject;
    static serializeBinaryToWriter(message: Appearance, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Appearance;
    static deserializeBinaryFromReader(message: Appearance, reader: jspb.BinaryReader): Appearance;
  }

  export namespace Appearance {
    export type AsObject = {
      title: string;
      description: string;
      moreMap: Array<[string, string]>;
    };
  }


  export class Location extends jspb.Message {
    getTitle(): string;
    setTitle(value: string): Location;

    getDescription(): string;
    setDescription(value: string): Location;

    getArchitectureReference(): string;
    setArchitectureReference(value: string): Location;

    getFloor(): string;
    setFloor(value: string): Location;

    getZone(): string;
    setZone(value: string): Location;

    getMoreMap(): jspb.Map<string, string>;
    clearMoreMap(): Location;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Location.AsObject;
    static toObject(includeInstance: boolean, msg: Location): Location.AsObject;
    static serializeBinaryToWriter(message: Location, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Location;
    static deserializeBinaryFromReader(message: Location, reader: jspb.BinaryReader): Location;
  }

  export namespace Location {
    export type AsObject = {
      title: string;
      description: string;
      architectureReference: string;
      floor: string;
      zone: string;
      moreMap: Array<[string, string]>;
    };
  }


  export class ID extends jspb.Message {
    getSerialNumber(): string;
    setSerialNumber(value: string): ID;

    getBim(): string;
    setBim(value: string): ID;

    getBacnet(): string;
    setBacnet(value: string): ID;

    getMoreMap(): jspb.Map<string, string>;
    clearMoreMap(): ID;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ID.AsObject;
    static toObject(includeInstance: boolean, msg: ID): ID.AsObject;
    static serializeBinaryToWriter(message: ID, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ID;
    static deserializeBinaryFromReader(message: ID, reader: jspb.BinaryReader): ID;
  }

  export namespace ID {
    export type AsObject = {
      serialNumber: string;
      bim: string;
      bacnet: string;
      moreMap: Array<[string, string]>;
    };
  }


  export class Product extends jspb.Message {
    getTitle(): string;
    setTitle(value: string): Product;

    getManufacturer(): string;
    setManufacturer(value: string): Product;

    getModel(): string;
    setModel(value: string): Product;

    getHardwareVersion(): string;
    setHardwareVersion(value: string): Product;

    getFirmwareVersion(): string;
    setFirmwareVersion(value: string): Product;

    getSoftwareVersion(): string;
    setSoftwareVersion(value: string): Product;

    getKind(): Metadata.Product.Kind | undefined;
    setKind(value?: Metadata.Product.Kind): Product;
    hasKind(): boolean;
    clearKind(): Product;

    getMoreMap(): jspb.Map<string, string>;
    clearMoreMap(): Product;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Product.AsObject;
    static toObject(includeInstance: boolean, msg: Product): Product.AsObject;
    static serializeBinaryToWriter(message: Product, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Product;
    static deserializeBinaryFromReader(message: Product, reader: jspb.BinaryReader): Product;
  }

  export namespace Product {
    export type AsObject = {
      title: string;
      manufacturer: string;
      model: string;
      hardwareVersion: string;
      firmwareVersion: string;
      softwareVersion: string;
      kind?: Metadata.Product.Kind.AsObject;
      moreMap: Array<[string, string]>;
    };

    export class Kind extends jspb.Message {
      getTitle(): string;
      setTitle(value: string): Kind;

      getCode(): string;
      setCode(value: string): Kind;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): Kind.AsObject;
      static toObject(includeInstance: boolean, msg: Kind): Kind.AsObject;
      static serializeBinaryToWriter(message: Kind, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): Kind;
      static deserializeBinaryFromReader(message: Kind, reader: jspb.BinaryReader): Kind;
    }

    export namespace Kind {
      export type AsObject = {
        title: string;
        code: string;
      };
    }

  }


  export class Revision extends jspb.Message {
    getTitle(): string;
    setTitle(value: string): Revision;

    getManufactureDate(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setManufactureDate(value?: google_protobuf_timestamp_pb.Timestamp): Revision;
    hasManufactureDate(): boolean;
    clearManufactureDate(): Revision;

    getModel(): string;
    setModel(value: string): Revision;

    getHardwareVersion(): string;
    setHardwareVersion(value: string): Revision;

    getFirmwareVersion(): string;
    setFirmwareVersion(value: string): Revision;

    getSoftwareVersion(): string;
    setSoftwareVersion(value: string): Revision;

    getBatch(): string;
    setBatch(value: string): Revision;

    getMoreMap(): jspb.Map<string, string>;
    clearMoreMap(): Revision;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Revision.AsObject;
    static toObject(includeInstance: boolean, msg: Revision): Revision.AsObject;
    static serializeBinaryToWriter(message: Revision, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Revision;
    static deserializeBinaryFromReader(message: Revision, reader: jspb.BinaryReader): Revision;
  }

  export namespace Revision {
    export type AsObject = {
      title: string;
      manufactureDate?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      model: string;
      hardwareVersion: string;
      firmwareVersion: string;
      softwareVersion: string;
      batch: string;
      moreMap: Array<[string, string]>;
    };
  }


  export class Installation extends jspb.Message {
    getInstallTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setInstallTime(value?: google_protobuf_timestamp_pb.Timestamp): Installation;
    hasInstallTime(): boolean;
    clearInstallTime(): Installation;

    getReplaceTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setReplaceTime(value?: google_protobuf_timestamp_pb.Timestamp): Installation;
    hasReplaceTime(): boolean;
    clearReplaceTime(): Installation;

    getInstaller(): string;
    setInstaller(value: string): Installation;

    getLabelled(): boolean;
    setLabelled(value: boolean): Installation;

    getLabelTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setLabelTime(value?: google_protobuf_timestamp_pb.Timestamp): Installation;
    hasLabelTime(): boolean;
    clearLabelTime(): Installation;

    getMoreMap(): jspb.Map<string, string>;
    clearMoreMap(): Installation;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Installation.AsObject;
    static toObject(includeInstance: boolean, msg: Installation): Installation.AsObject;
    static serializeBinaryToWriter(message: Installation, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Installation;
    static deserializeBinaryFromReader(message: Installation, reader: jspb.BinaryReader): Installation;
  }

  export namespace Installation {
    export type AsObject = {
      installTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      replaceTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      installer: string;
      labelled: boolean;
      labelTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      moreMap: Array<[string, string]>;
    };
  }


  export class NIC extends jspb.Message {
    getDisplayName(): string;
    setDisplayName(value: string): NIC;

    getMacAddress(): string;
    setMacAddress(value: string): NIC;

    getIp(): string;
    setIp(value: string): NIC;

    getNetwork(): string;
    setNetwork(value: string): NIC;

    getGateway(): string;
    setGateway(value: string): NIC;

    getDnsList(): Array<string>;
    setDnsList(value: Array<string>): NIC;
    clearDnsList(): NIC;
    addDns(value: string, index?: number): NIC;

    getAssignment(): Metadata.NIC.Assignment;
    setAssignment(value: Metadata.NIC.Assignment): NIC;

    getMoreMap(): jspb.Map<string, string>;
    clearMoreMap(): NIC;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): NIC.AsObject;
    static toObject(includeInstance: boolean, msg: NIC): NIC.AsObject;
    static serializeBinaryToWriter(message: NIC, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): NIC;
    static deserializeBinaryFromReader(message: NIC, reader: jspb.BinaryReader): NIC;
  }

  export namespace NIC {
    export type AsObject = {
      displayName: string;
      macAddress: string;
      ip: string;
      network: string;
      gateway: string;
      dnsList: Array<string>;
      assignment: Metadata.NIC.Assignment;
      moreMap: Array<[string, string]>;
    };

    export enum Assignment {
      ASSIGNMENT_UNSPECIFIED = 0,
      DHCP = 1,
      STATIC = 2,
    }
  }


  export class Membership extends jspb.Message {
    getGroup(): string;
    setGroup(value: string): Membership;

    getSubsystem(): string;
    setSubsystem(value: string): Membership;

    getMoreMap(): jspb.Map<string, string>;
    clearMoreMap(): Membership;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Membership.AsObject;
    static toObject(includeInstance: boolean, msg: Membership): Membership.AsObject;
    static serializeBinaryToWriter(message: Membership, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Membership;
    static deserializeBinaryFromReader(message: Membership, reader: jspb.BinaryReader): Membership;
  }

  export namespace Membership {
    export type AsObject = {
      group: string;
      subsystem: string;
      moreMap: Array<[string, string]>;
    };
  }

}

export class TraitMetadata extends jspb.Message {
  getName(): string;
  setName(value: string): TraitMetadata;

  getMoreMap(): jspb.Map<string, string>;
  clearMoreMap(): TraitMetadata;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TraitMetadata.AsObject;
  static toObject(includeInstance: boolean, msg: TraitMetadata): TraitMetadata.AsObject;
  static serializeBinaryToWriter(message: TraitMetadata, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TraitMetadata;
  static deserializeBinaryFromReader(message: TraitMetadata, reader: jspb.BinaryReader): TraitMetadata;
}

export namespace TraitMetadata {
  export type AsObject = {
    name: string;
    moreMap: Array<[string, string]>;
  };
}

export class GetMetadataRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetMetadataRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetMetadataRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetMetadataRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetMetadataRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetMetadataRequest): GetMetadataRequest.AsObject;
  static serializeBinaryToWriter(message: GetMetadataRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetMetadataRequest;
  static deserializeBinaryFromReader(message: GetMetadataRequest, reader: jspb.BinaryReader): GetMetadataRequest;
}

export namespace GetMetadataRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullMetadataRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullMetadataRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullMetadataRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullMetadataRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullMetadataRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullMetadataRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullMetadataRequest): PullMetadataRequest.AsObject;
  static serializeBinaryToWriter(message: PullMetadataRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullMetadataRequest;
  static deserializeBinaryFromReader(message: PullMetadataRequest, reader: jspb.BinaryReader): PullMetadataRequest;
}

export namespace PullMetadataRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullMetadataResponse extends jspb.Message {
  getChangesList(): Array<PullMetadataResponse.Change>;
  setChangesList(value: Array<PullMetadataResponse.Change>): PullMetadataResponse;
  clearChangesList(): PullMetadataResponse;
  addChanges(value?: PullMetadataResponse.Change, index?: number): PullMetadataResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullMetadataResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullMetadataResponse): PullMetadataResponse.AsObject;
  static serializeBinaryToWriter(message: PullMetadataResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullMetadataResponse;
  static deserializeBinaryFromReader(message: PullMetadataResponse, reader: jspb.BinaryReader): PullMetadataResponse;
}

export namespace PullMetadataResponse {
  export type AsObject = {
    changesList: Array<PullMetadataResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getMetadata(): Metadata | undefined;
    setMetadata(value?: Metadata): Change;
    hasMetadata(): boolean;
    clearMetadata(): Change;

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
      metadata?: Metadata.AsObject;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };
  }

}

