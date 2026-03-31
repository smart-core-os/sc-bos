import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"


export class Tenant extends jspb.Message {
  getId(): string;
  setId(value: string): Tenant;

  getTitle(): string;
  setTitle(value: string): Tenant;

  getCreateTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreateTime(value?: google_protobuf_timestamp_pb.Timestamp): Tenant;
  hasCreateTime(): boolean;
  clearCreateTime(): Tenant;

  getZoneNamesList(): Array<string>;
  setZoneNamesList(value: Array<string>): Tenant;
  clearZoneNamesList(): Tenant;
  addZoneNames(value: string, index?: number): Tenant;

  getEtag(): string;
  setEtag(value: string): Tenant;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Tenant.AsObject;
  static toObject(includeInstance: boolean, msg: Tenant): Tenant.AsObject;
  static serializeBinaryToWriter(message: Tenant, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Tenant;
  static deserializeBinaryFromReader(message: Tenant, reader: jspb.BinaryReader): Tenant;
}

export namespace Tenant {
  export type AsObject = {
    id: string;
    title: string;
    createTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    zoneNamesList: Array<string>;
    etag: string;
  };
}

export class Secret extends jspb.Message {
  getId(): string;
  setId(value: string): Secret;

  getTenant(): Tenant | undefined;
  setTenant(value?: Tenant): Secret;
  hasTenant(): boolean;
  clearTenant(): Secret;

  getSecretHash(): Uint8Array | string;
  getSecretHash_asU8(): Uint8Array;
  getSecretHash_asB64(): string;
  setSecretHash(value: Uint8Array | string): Secret;

  getSecret(): string;
  setSecret(value: string): Secret;

  getNote(): string;
  setNote(value: string): Secret;

  getCreateTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreateTime(value?: google_protobuf_timestamp_pb.Timestamp): Secret;
  hasCreateTime(): boolean;
  clearCreateTime(): Secret;

  getExpireTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setExpireTime(value?: google_protobuf_timestamp_pb.Timestamp): Secret;
  hasExpireTime(): boolean;
  clearExpireTime(): Secret;

  getFirstUseTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setFirstUseTime(value?: google_protobuf_timestamp_pb.Timestamp): Secret;
  hasFirstUseTime(): boolean;
  clearFirstUseTime(): Secret;

  getLastUseTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastUseTime(value?: google_protobuf_timestamp_pb.Timestamp): Secret;
  hasLastUseTime(): boolean;
  clearLastUseTime(): Secret;

  getEtag(): string;
  setEtag(value: string): Secret;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Secret.AsObject;
  static toObject(includeInstance: boolean, msg: Secret): Secret.AsObject;
  static serializeBinaryToWriter(message: Secret, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Secret;
  static deserializeBinaryFromReader(message: Secret, reader: jspb.BinaryReader): Secret;
}

export namespace Secret {
  export type AsObject = {
    id: string;
    tenant?: Tenant.AsObject;
    secretHash: Uint8Array | string;
    secret: string;
    note: string;
    createTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    expireTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    firstUseTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    lastUseTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    etag: string;
  };
}

export class ListTenantsRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListTenantsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListTenantsRequest): ListTenantsRequest.AsObject;
  static serializeBinaryToWriter(message: ListTenantsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListTenantsRequest;
  static deserializeBinaryFromReader(message: ListTenantsRequest, reader: jspb.BinaryReader): ListTenantsRequest;
}

export namespace ListTenantsRequest {
  export type AsObject = {
  };
}

export class ListTenantsResponse extends jspb.Message {
  getTenantsList(): Array<Tenant>;
  setTenantsList(value: Array<Tenant>): ListTenantsResponse;
  clearTenantsList(): ListTenantsResponse;
  addTenants(value?: Tenant, index?: number): Tenant;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListTenantsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListTenantsResponse): ListTenantsResponse.AsObject;
  static serializeBinaryToWriter(message: ListTenantsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListTenantsResponse;
  static deserializeBinaryFromReader(message: ListTenantsResponse, reader: jspb.BinaryReader): ListTenantsResponse;
}

export namespace ListTenantsResponse {
  export type AsObject = {
    tenantsList: Array<Tenant.AsObject>;
  };
}

export class PullTenantsRequest extends jspb.Message {
  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullTenantsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullTenantsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullTenantsRequest): PullTenantsRequest.AsObject;
  static serializeBinaryToWriter(message: PullTenantsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullTenantsRequest;
  static deserializeBinaryFromReader(message: PullTenantsRequest, reader: jspb.BinaryReader): PullTenantsRequest;
}

export namespace PullTenantsRequest {
  export type AsObject = {
    updatesOnly: boolean;
  };
}

export class PullTenantsResponse extends jspb.Message {
  getChangesList(): Array<PullTenantsResponse.Change>;
  setChangesList(value: Array<PullTenantsResponse.Change>): PullTenantsResponse;
  clearChangesList(): PullTenantsResponse;
  addChanges(value?: PullTenantsResponse.Change, index?: number): PullTenantsResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullTenantsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullTenantsResponse): PullTenantsResponse.AsObject;
  static serializeBinaryToWriter(message: PullTenantsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullTenantsResponse;
  static deserializeBinaryFromReader(message: PullTenantsResponse, reader: jspb.BinaryReader): PullTenantsResponse;
}

export namespace PullTenantsResponse {
  export type AsObject = {
    changesList: Array<PullTenantsResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getTenant(): Tenant | undefined;
    setTenant(value?: Tenant): Change;
    hasTenant(): boolean;
    clearTenant(): Change;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Change.AsObject;
    static toObject(includeInstance: boolean, msg: Change): Change.AsObject;
    static serializeBinaryToWriter(message: Change, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Change;
    static deserializeBinaryFromReader(message: Change, reader: jspb.BinaryReader): Change;
  }

  export namespace Change {
    export type AsObject = {
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      tenant?: Tenant.AsObject;
    };
  }

}

export class CreateTenantRequest extends jspb.Message {
  getTenant(): Tenant | undefined;
  setTenant(value?: Tenant): CreateTenantRequest;
  hasTenant(): boolean;
  clearTenant(): CreateTenantRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateTenantRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateTenantRequest): CreateTenantRequest.AsObject;
  static serializeBinaryToWriter(message: CreateTenantRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateTenantRequest;
  static deserializeBinaryFromReader(message: CreateTenantRequest, reader: jspb.BinaryReader): CreateTenantRequest;
}

export namespace CreateTenantRequest {
  export type AsObject = {
    tenant?: Tenant.AsObject;
  };
}

export class GetTenantRequest extends jspb.Message {
  getId(): string;
  setId(value: string): GetTenantRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetTenantRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetTenantRequest): GetTenantRequest.AsObject;
  static serializeBinaryToWriter(message: GetTenantRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetTenantRequest;
  static deserializeBinaryFromReader(message: GetTenantRequest, reader: jspb.BinaryReader): GetTenantRequest;
}

export namespace GetTenantRequest {
  export type AsObject = {
    id: string;
  };
}

export class UpdateTenantRequest extends jspb.Message {
  getTenant(): Tenant | undefined;
  setTenant(value?: Tenant): UpdateTenantRequest;
  hasTenant(): boolean;
  clearTenant(): UpdateTenantRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateTenantRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateTenantRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateTenantRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateTenantRequest): UpdateTenantRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateTenantRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateTenantRequest;
  static deserializeBinaryFromReader(message: UpdateTenantRequest, reader: jspb.BinaryReader): UpdateTenantRequest;
}

export namespace UpdateTenantRequest {
  export type AsObject = {
    tenant?: Tenant.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class DeleteTenantRequest extends jspb.Message {
  getId(): string;
  setId(value: string): DeleteTenantRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteTenantRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteTenantRequest): DeleteTenantRequest.AsObject;
  static serializeBinaryToWriter(message: DeleteTenantRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteTenantRequest;
  static deserializeBinaryFromReader(message: DeleteTenantRequest, reader: jspb.BinaryReader): DeleteTenantRequest;
}

export namespace DeleteTenantRequest {
  export type AsObject = {
    id: string;
  };
}

export class DeleteTenantResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteTenantResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteTenantResponse): DeleteTenantResponse.AsObject;
  static serializeBinaryToWriter(message: DeleteTenantResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteTenantResponse;
  static deserializeBinaryFromReader(message: DeleteTenantResponse, reader: jspb.BinaryReader): DeleteTenantResponse;
}

export namespace DeleteTenantResponse {
  export type AsObject = {
  };
}

export class PullTenantRequest extends jspb.Message {
  getId(): string;
  setId(value: string): PullTenantRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullTenantRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullTenantRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullTenantRequest): PullTenantRequest.AsObject;
  static serializeBinaryToWriter(message: PullTenantRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullTenantRequest;
  static deserializeBinaryFromReader(message: PullTenantRequest, reader: jspb.BinaryReader): PullTenantRequest;
}

export namespace PullTenantRequest {
  export type AsObject = {
    id: string;
    updatesOnly: boolean;
  };
}

export class PullTenantResponse extends jspb.Message {
  getChangesList(): Array<PullTenantResponse.Change>;
  setChangesList(value: Array<PullTenantResponse.Change>): PullTenantResponse;
  clearChangesList(): PullTenantResponse;
  addChanges(value?: PullTenantResponse.Change, index?: number): PullTenantResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullTenantResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullTenantResponse): PullTenantResponse.AsObject;
  static serializeBinaryToWriter(message: PullTenantResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullTenantResponse;
  static deserializeBinaryFromReader(message: PullTenantResponse, reader: jspb.BinaryReader): PullTenantResponse;
}

export namespace PullTenantResponse {
  export type AsObject = {
    changesList: Array<PullTenantResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getTenant(): Tenant | undefined;
    setTenant(value?: Tenant): Change;
    hasTenant(): boolean;
    clearTenant(): Change;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Change.AsObject;
    static toObject(includeInstance: boolean, msg: Change): Change.AsObject;
    static serializeBinaryToWriter(message: Change, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Change;
    static deserializeBinaryFromReader(message: Change, reader: jspb.BinaryReader): Change;
  }

  export namespace Change {
    export type AsObject = {
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      tenant?: Tenant.AsObject;
    };
  }

}

export class AddTenantZonesRequest extends jspb.Message {
  getTenantId(): string;
  setTenantId(value: string): AddTenantZonesRequest;

  getAddZoneNamesList(): Array<string>;
  setAddZoneNamesList(value: Array<string>): AddTenantZonesRequest;
  clearAddZoneNamesList(): AddTenantZonesRequest;
  addAddZoneNames(value: string, index?: number): AddTenantZonesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AddTenantZonesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: AddTenantZonesRequest): AddTenantZonesRequest.AsObject;
  static serializeBinaryToWriter(message: AddTenantZonesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AddTenantZonesRequest;
  static deserializeBinaryFromReader(message: AddTenantZonesRequest, reader: jspb.BinaryReader): AddTenantZonesRequest;
}

export namespace AddTenantZonesRequest {
  export type AsObject = {
    tenantId: string;
    addZoneNamesList: Array<string>;
  };
}

export class RemoveTenantZonesRequest extends jspb.Message {
  getTenantId(): string;
  setTenantId(value: string): RemoveTenantZonesRequest;

  getRemoveZoneNamesList(): Array<string>;
  setRemoveZoneNamesList(value: Array<string>): RemoveTenantZonesRequest;
  clearRemoveZoneNamesList(): RemoveTenantZonesRequest;
  addRemoveZoneNames(value: string, index?: number): RemoveTenantZonesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RemoveTenantZonesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RemoveTenantZonesRequest): RemoveTenantZonesRequest.AsObject;
  static serializeBinaryToWriter(message: RemoveTenantZonesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RemoveTenantZonesRequest;
  static deserializeBinaryFromReader(message: RemoveTenantZonesRequest, reader: jspb.BinaryReader): RemoveTenantZonesRequest;
}

export namespace RemoveTenantZonesRequest {
  export type AsObject = {
    tenantId: string;
    removeZoneNamesList: Array<string>;
  };
}

export class ListSecretsRequest extends jspb.Message {
  getIncludeHash(): boolean;
  setIncludeHash(value: boolean): ListSecretsRequest;

  getFilter(): string;
  setFilter(value: string): ListSecretsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListSecretsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListSecretsRequest): ListSecretsRequest.AsObject;
  static serializeBinaryToWriter(message: ListSecretsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListSecretsRequest;
  static deserializeBinaryFromReader(message: ListSecretsRequest, reader: jspb.BinaryReader): ListSecretsRequest;
}

export namespace ListSecretsRequest {
  export type AsObject = {
    includeHash: boolean;
    filter: string;
  };
}

export class ListSecretsResponse extends jspb.Message {
  getSecretsList(): Array<Secret>;
  setSecretsList(value: Array<Secret>): ListSecretsResponse;
  clearSecretsList(): ListSecretsResponse;
  addSecrets(value?: Secret, index?: number): Secret;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListSecretsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListSecretsResponse): ListSecretsResponse.AsObject;
  static serializeBinaryToWriter(message: ListSecretsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListSecretsResponse;
  static deserializeBinaryFromReader(message: ListSecretsResponse, reader: jspb.BinaryReader): ListSecretsResponse;
}

export namespace ListSecretsResponse {
  export type AsObject = {
    secretsList: Array<Secret.AsObject>;
  };
}

export class PullSecretsRequest extends jspb.Message {
  getIncludeHash(): boolean;
  setIncludeHash(value: boolean): PullSecretsRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullSecretsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullSecretsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullSecretsRequest): PullSecretsRequest.AsObject;
  static serializeBinaryToWriter(message: PullSecretsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullSecretsRequest;
  static deserializeBinaryFromReader(message: PullSecretsRequest, reader: jspb.BinaryReader): PullSecretsRequest;
}

export namespace PullSecretsRequest {
  export type AsObject = {
    includeHash: boolean;
    updatesOnly: boolean;
  };
}

export class PullSecretsResponse extends jspb.Message {
  getChangesList(): Array<PullSecretsResponse.Change>;
  setChangesList(value: Array<PullSecretsResponse.Change>): PullSecretsResponse;
  clearChangesList(): PullSecretsResponse;
  addChanges(value?: PullSecretsResponse.Change, index?: number): PullSecretsResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullSecretsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullSecretsResponse): PullSecretsResponse.AsObject;
  static serializeBinaryToWriter(message: PullSecretsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullSecretsResponse;
  static deserializeBinaryFromReader(message: PullSecretsResponse, reader: jspb.BinaryReader): PullSecretsResponse;
}

export namespace PullSecretsResponse {
  export type AsObject = {
    changesList: Array<PullSecretsResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getSecret(): Secret | undefined;
    setSecret(value?: Secret): Change;
    hasSecret(): boolean;
    clearSecret(): Change;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Change.AsObject;
    static toObject(includeInstance: boolean, msg: Change): Change.AsObject;
    static serializeBinaryToWriter(message: Change, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Change;
    static deserializeBinaryFromReader(message: Change, reader: jspb.BinaryReader): Change;
  }

  export namespace Change {
    export type AsObject = {
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      secret?: Secret.AsObject;
    };
  }

}

export class CreateSecretRequest extends jspb.Message {
  getSecret(): Secret | undefined;
  setSecret(value?: Secret): CreateSecretRequest;
  hasSecret(): boolean;
  clearSecret(): CreateSecretRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateSecretRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateSecretRequest): CreateSecretRequest.AsObject;
  static serializeBinaryToWriter(message: CreateSecretRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateSecretRequest;
  static deserializeBinaryFromReader(message: CreateSecretRequest, reader: jspb.BinaryReader): CreateSecretRequest;
}

export namespace CreateSecretRequest {
  export type AsObject = {
    secret?: Secret.AsObject;
  };
}

export class VerifySecretRequest extends jspb.Message {
  getTenantId(): string;
  setTenantId(value: string): VerifySecretRequest;

  getSecret(): string;
  setSecret(value: string): VerifySecretRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): VerifySecretRequest.AsObject;
  static toObject(includeInstance: boolean, msg: VerifySecretRequest): VerifySecretRequest.AsObject;
  static serializeBinaryToWriter(message: VerifySecretRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): VerifySecretRequest;
  static deserializeBinaryFromReader(message: VerifySecretRequest, reader: jspb.BinaryReader): VerifySecretRequest;
}

export namespace VerifySecretRequest {
  export type AsObject = {
    tenantId: string;
    secret: string;
  };
}

export class GetSecretRequest extends jspb.Message {
  getId(): string;
  setId(value: string): GetSecretRequest;

  getIncludeHash(): boolean;
  setIncludeHash(value: boolean): GetSecretRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetSecretRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetSecretRequest): GetSecretRequest.AsObject;
  static serializeBinaryToWriter(message: GetSecretRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetSecretRequest;
  static deserializeBinaryFromReader(message: GetSecretRequest, reader: jspb.BinaryReader): GetSecretRequest;
}

export namespace GetSecretRequest {
  export type AsObject = {
    id: string;
    includeHash: boolean;
  };
}

export class GetSecretByHashRequest extends jspb.Message {
  getSecretHash(): Uint8Array | string;
  getSecretHash_asU8(): Uint8Array;
  getSecretHash_asB64(): string;
  setSecretHash(value: Uint8Array | string): GetSecretByHashRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetSecretByHashRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetSecretByHashRequest): GetSecretByHashRequest.AsObject;
  static serializeBinaryToWriter(message: GetSecretByHashRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetSecretByHashRequest;
  static deserializeBinaryFromReader(message: GetSecretByHashRequest, reader: jspb.BinaryReader): GetSecretByHashRequest;
}

export namespace GetSecretByHashRequest {
  export type AsObject = {
    secretHash: Uint8Array | string;
  };
}

export class UpdateSecretRequest extends jspb.Message {
  getSecret(): Secret | undefined;
  setSecret(value?: Secret): UpdateSecretRequest;
  hasSecret(): boolean;
  clearSecret(): UpdateSecretRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateSecretRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateSecretRequest): UpdateSecretRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateSecretRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateSecretRequest;
  static deserializeBinaryFromReader(message: UpdateSecretRequest, reader: jspb.BinaryReader): UpdateSecretRequest;
}

export namespace UpdateSecretRequest {
  export type AsObject = {
    secret?: Secret.AsObject;
  };
}

export class DeleteSecretRequest extends jspb.Message {
  getId(): string;
  setId(value: string): DeleteSecretRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteSecretRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteSecretRequest): DeleteSecretRequest.AsObject;
  static serializeBinaryToWriter(message: DeleteSecretRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteSecretRequest;
  static deserializeBinaryFromReader(message: DeleteSecretRequest, reader: jspb.BinaryReader): DeleteSecretRequest;
}

export namespace DeleteSecretRequest {
  export type AsObject = {
    id: string;
  };
}

export class DeleteSecretResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteSecretResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteSecretResponse): DeleteSecretResponse.AsObject;
  static serializeBinaryToWriter(message: DeleteSecretResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteSecretResponse;
  static deserializeBinaryFromReader(message: DeleteSecretResponse, reader: jspb.BinaryReader): DeleteSecretResponse;
}

export namespace DeleteSecretResponse {
  export type AsObject = {
  };
}

export class PullSecretRequest extends jspb.Message {
  getId(): string;
  setId(value: string): PullSecretRequest;

  getIncludeHash(): boolean;
  setIncludeHash(value: boolean): PullSecretRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullSecretRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullSecretRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullSecretRequest): PullSecretRequest.AsObject;
  static serializeBinaryToWriter(message: PullSecretRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullSecretRequest;
  static deserializeBinaryFromReader(message: PullSecretRequest, reader: jspb.BinaryReader): PullSecretRequest;
}

export namespace PullSecretRequest {
  export type AsObject = {
    id: string;
    includeHash: boolean;
    updatesOnly: boolean;
  };
}

export class PullSecretResponse extends jspb.Message {
  getChangesList(): Array<PullSecretResponse.Change>;
  setChangesList(value: Array<PullSecretResponse.Change>): PullSecretResponse;
  clearChangesList(): PullSecretResponse;
  addChanges(value?: PullSecretResponse.Change, index?: number): PullSecretResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullSecretResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullSecretResponse): PullSecretResponse.AsObject;
  static serializeBinaryToWriter(message: PullSecretResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullSecretResponse;
  static deserializeBinaryFromReader(message: PullSecretResponse, reader: jspb.BinaryReader): PullSecretResponse;
}

export namespace PullSecretResponse {
  export type AsObject = {
    changesList: Array<PullSecretResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getSecret(): Secret | undefined;
    setSecret(value?: Secret): Change;
    hasSecret(): boolean;
    clearSecret(): Change;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Change.AsObject;
    static toObject(includeInstance: boolean, msg: Change): Change.AsObject;
    static serializeBinaryToWriter(message: Change, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Change;
    static deserializeBinaryFromReader(message: Change, reader: jspb.BinaryReader): Change;
  }

  export namespace Change {
    export type AsObject = {
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      secret?: Secret.AsObject;
    };
  }

}

export class RegenerateSecretRequest extends jspb.Message {
  getId(): string;
  setId(value: string): RegenerateSecretRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RegenerateSecretRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RegenerateSecretRequest): RegenerateSecretRequest.AsObject;
  static serializeBinaryToWriter(message: RegenerateSecretRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RegenerateSecretRequest;
  static deserializeBinaryFromReader(message: RegenerateSecretRequest, reader: jspb.BinaryReader): RegenerateSecretRequest;
}

export namespace RegenerateSecretRequest {
  export type AsObject = {
    id: string;
  };
}

