import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_change_pb from '@smart-core-os/sc-api-grpc-web/types/change_pb'; // proto import: "types/change.proto"


export class Service extends jspb.Message {
  getId(): string;
  setId(value: string): Service;

  getType(): string;
  setType(value: string): Service;

  getActive(): boolean;
  setActive(value: boolean): Service;

  getLastInactiveTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastInactiveTime(value?: google_protobuf_timestamp_pb.Timestamp): Service;
  hasLastInactiveTime(): boolean;
  clearLastInactiveTime(): Service;

  getLastActiveTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastActiveTime(value?: google_protobuf_timestamp_pb.Timestamp): Service;
  hasLastActiveTime(): boolean;
  clearLastActiveTime(): Service;

  getLoading(): boolean;
  setLoading(value: boolean): Service;

  getLastLoadingStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastLoadingStartTime(value?: google_protobuf_timestamp_pb.Timestamp): Service;
  hasLastLoadingStartTime(): boolean;
  clearLastLoadingStartTime(): Service;

  getLastLoadingEndTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastLoadingEndTime(value?: google_protobuf_timestamp_pb.Timestamp): Service;
  hasLastLoadingEndTime(): boolean;
  clearLastLoadingEndTime(): Service;

  getError(): string;
  setError(value: string): Service;

  getLastErrorTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastErrorTime(value?: google_protobuf_timestamp_pb.Timestamp): Service;
  hasLastErrorTime(): boolean;
  clearLastErrorTime(): Service;

  getConfigRaw(): string;
  setConfigRaw(value: string): Service;

  getLastConfigTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastConfigTime(value?: google_protobuf_timestamp_pb.Timestamp): Service;
  hasLastConfigTime(): boolean;
  clearLastConfigTime(): Service;

  getFailedAttempts(): number;
  setFailedAttempts(value: number): Service;

  getNextAttemptTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setNextAttemptTime(value?: google_protobuf_timestamp_pb.Timestamp): Service;
  hasNextAttemptTime(): boolean;
  clearNextAttemptTime(): Service;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Service.AsObject;
  static toObject(includeInstance: boolean, msg: Service): Service.AsObject;
  static serializeBinaryToWriter(message: Service, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Service;
  static deserializeBinaryFromReader(message: Service, reader: jspb.BinaryReader): Service;
}

export namespace Service {
  export type AsObject = {
    id: string;
    type: string;
    active: boolean;
    lastInactiveTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    lastActiveTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    loading: boolean;
    lastLoadingStartTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    lastLoadingEndTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    error: string;
    lastErrorTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    configRaw: string;
    lastConfigTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    failedAttempts: number;
    nextAttemptTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class ServiceMetadata extends jspb.Message {
  getTotalCount(): number;
  setTotalCount(value: number): ServiceMetadata;

  getTypeCountsMap(): jspb.Map<string, number>;
  clearTypeCountsMap(): ServiceMetadata;

  getTotalActiveCount(): number;
  setTotalActiveCount(value: number): ServiceMetadata;

  getTotalErrorCount(): number;
  setTotalErrorCount(value: number): ServiceMetadata;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ServiceMetadata.AsObject;
  static toObject(includeInstance: boolean, msg: ServiceMetadata): ServiceMetadata.AsObject;
  static serializeBinaryToWriter(message: ServiceMetadata, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ServiceMetadata;
  static deserializeBinaryFromReader(message: ServiceMetadata, reader: jspb.BinaryReader): ServiceMetadata;
}

export namespace ServiceMetadata {
  export type AsObject = {
    totalCount: number;
    typeCountsMap: Array<[string, number]>;
    totalActiveCount: number;
    totalErrorCount: number;
  };
}

export class GetServiceRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetServiceRequest;

  getId(): string;
  setId(value: string): GetServiceRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetServiceRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetServiceRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetServiceRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetServiceRequest): GetServiceRequest.AsObject;
  static serializeBinaryToWriter(message: GetServiceRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetServiceRequest;
  static deserializeBinaryFromReader(message: GetServiceRequest, reader: jspb.BinaryReader): GetServiceRequest;
}

export namespace GetServiceRequest {
  export type AsObject = {
    name: string;
    id: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullServiceRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullServiceRequest;

  getId(): string;
  setId(value: string): PullServiceRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullServiceRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullServiceRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullServiceRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullServiceRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullServiceRequest): PullServiceRequest.AsObject;
  static serializeBinaryToWriter(message: PullServiceRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullServiceRequest;
  static deserializeBinaryFromReader(message: PullServiceRequest, reader: jspb.BinaryReader): PullServiceRequest;
}

export namespace PullServiceRequest {
  export type AsObject = {
    name: string;
    id: string;
    updatesOnly: boolean;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullServiceResponse extends jspb.Message {
  getChangesList(): Array<PullServiceResponse.Change>;
  setChangesList(value: Array<PullServiceResponse.Change>): PullServiceResponse;
  clearChangesList(): PullServiceResponse;
  addChanges(value?: PullServiceResponse.Change, index?: number): PullServiceResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullServiceResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullServiceResponse): PullServiceResponse.AsObject;
  static serializeBinaryToWriter(message: PullServiceResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullServiceResponse;
  static deserializeBinaryFromReader(message: PullServiceResponse, reader: jspb.BinaryReader): PullServiceResponse;
}

export namespace PullServiceResponse {
  export type AsObject = {
    changesList: Array<PullServiceResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getService(): Service | undefined;
    setService(value?: Service): Change;
    hasService(): boolean;
    clearService(): Change;

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
      service?: Service.AsObject;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };
  }

}

export class CreateServiceRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CreateServiceRequest;

  getService(): Service | undefined;
  setService(value?: Service): CreateServiceRequest;
  hasService(): boolean;
  clearService(): CreateServiceRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateServiceRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateServiceRequest): CreateServiceRequest.AsObject;
  static serializeBinaryToWriter(message: CreateServiceRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateServiceRequest;
  static deserializeBinaryFromReader(message: CreateServiceRequest, reader: jspb.BinaryReader): CreateServiceRequest;
}

export namespace CreateServiceRequest {
  export type AsObject = {
    name: string;
    service?: Service.AsObject;
  };
}

export class DeleteServiceRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DeleteServiceRequest;

  getId(): string;
  setId(value: string): DeleteServiceRequest;

  getAllowMissing(): boolean;
  setAllowMissing(value: boolean): DeleteServiceRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeleteServiceRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeleteServiceRequest): DeleteServiceRequest.AsObject;
  static serializeBinaryToWriter(message: DeleteServiceRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeleteServiceRequest;
  static deserializeBinaryFromReader(message: DeleteServiceRequest, reader: jspb.BinaryReader): DeleteServiceRequest;
}

export namespace DeleteServiceRequest {
  export type AsObject = {
    name: string;
    id: string;
    allowMissing: boolean;
  };
}

export class ListServicesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListServicesRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListServicesRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListServicesRequest;

  getPageSize(): number;
  setPageSize(value: number): ListServicesRequest;

  getPageToken(): string;
  setPageToken(value: string): ListServicesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListServicesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListServicesRequest): ListServicesRequest.AsObject;
  static serializeBinaryToWriter(message: ListServicesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListServicesRequest;
  static deserializeBinaryFromReader(message: ListServicesRequest, reader: jspb.BinaryReader): ListServicesRequest;
}

export namespace ListServicesRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    pageSize: number;
    pageToken: string;
  };
}

export class ListServicesResponse extends jspb.Message {
  getServicesList(): Array<Service>;
  setServicesList(value: Array<Service>): ListServicesResponse;
  clearServicesList(): ListServicesResponse;
  addServices(value?: Service, index?: number): Service;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListServicesResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListServicesResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListServicesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListServicesResponse): ListServicesResponse.AsObject;
  static serializeBinaryToWriter(message: ListServicesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListServicesResponse;
  static deserializeBinaryFromReader(message: ListServicesResponse, reader: jspb.BinaryReader): ListServicesResponse;
}

export namespace ListServicesResponse {
  export type AsObject = {
    servicesList: Array<Service.AsObject>;
    nextPageToken: string;
    totalSize: number;
  };
}

export class PullServicesRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullServicesRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullServicesRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullServicesRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullServicesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullServicesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullServicesRequest): PullServicesRequest.AsObject;
  static serializeBinaryToWriter(message: PullServicesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullServicesRequest;
  static deserializeBinaryFromReader(message: PullServicesRequest, reader: jspb.BinaryReader): PullServicesRequest;
}

export namespace PullServicesRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullServicesResponse extends jspb.Message {
  getChangesList(): Array<PullServicesResponse.Change>;
  setChangesList(value: Array<PullServicesResponse.Change>): PullServicesResponse;
  clearChangesList(): PullServicesResponse;
  addChanges(value?: PullServicesResponse.Change, index?: number): PullServicesResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullServicesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullServicesResponse): PullServicesResponse.AsObject;
  static serializeBinaryToWriter(message: PullServicesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullServicesResponse;
  static deserializeBinaryFromReader(message: PullServicesResponse, reader: jspb.BinaryReader): PullServicesResponse;
}

export namespace PullServicesResponse {
  export type AsObject = {
    changesList: Array<PullServicesResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getType(): types_change_pb.ChangeType;
    setType(value: types_change_pb.ChangeType): Change;

    getNewValue(): Service | undefined;
    setNewValue(value?: Service): Change;
    hasNewValue(): boolean;
    clearNewValue(): Change;

    getOldValue(): Service | undefined;
    setOldValue(value?: Service): Change;
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
      newValue?: Service.AsObject;
      oldValue?: Service.AsObject;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };
  }

}

export class StartServiceRequest extends jspb.Message {
  getName(): string;
  setName(value: string): StartServiceRequest;

  getId(): string;
  setId(value: string): StartServiceRequest;

  getAllowActive(): boolean;
  setAllowActive(value: boolean): StartServiceRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StartServiceRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StartServiceRequest): StartServiceRequest.AsObject;
  static serializeBinaryToWriter(message: StartServiceRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StartServiceRequest;
  static deserializeBinaryFromReader(message: StartServiceRequest, reader: jspb.BinaryReader): StartServiceRequest;
}

export namespace StartServiceRequest {
  export type AsObject = {
    name: string;
    id: string;
    allowActive: boolean;
  };
}

export class ConfigureServiceRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ConfigureServiceRequest;

  getId(): string;
  setId(value: string): ConfigureServiceRequest;

  getConfigRaw(): string;
  setConfigRaw(value: string): ConfigureServiceRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ConfigureServiceRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ConfigureServiceRequest): ConfigureServiceRequest.AsObject;
  static serializeBinaryToWriter(message: ConfigureServiceRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ConfigureServiceRequest;
  static deserializeBinaryFromReader(message: ConfigureServiceRequest, reader: jspb.BinaryReader): ConfigureServiceRequest;
}

export namespace ConfigureServiceRequest {
  export type AsObject = {
    name: string;
    id: string;
    configRaw: string;
  };
}

export class StopServiceRequest extends jspb.Message {
  getName(): string;
  setName(value: string): StopServiceRequest;

  getId(): string;
  setId(value: string): StopServiceRequest;

  getAllowInactive(): boolean;
  setAllowInactive(value: boolean): StopServiceRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StopServiceRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StopServiceRequest): StopServiceRequest.AsObject;
  static serializeBinaryToWriter(message: StopServiceRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StopServiceRequest;
  static deserializeBinaryFromReader(message: StopServiceRequest, reader: jspb.BinaryReader): StopServiceRequest;
}

export namespace StopServiceRequest {
  export type AsObject = {
    name: string;
    id: string;
    allowInactive: boolean;
  };
}

export class GetServiceMetadataRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetServiceMetadataRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetServiceMetadataRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetServiceMetadataRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetServiceMetadataRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetServiceMetadataRequest): GetServiceMetadataRequest.AsObject;
  static serializeBinaryToWriter(message: GetServiceMetadataRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetServiceMetadataRequest;
  static deserializeBinaryFromReader(message: GetServiceMetadataRequest, reader: jspb.BinaryReader): GetServiceMetadataRequest;
}

export namespace GetServiceMetadataRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullServiceMetadataRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullServiceMetadataRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullServiceMetadataRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullServiceMetadataRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullServiceMetadataRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullServiceMetadataRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullServiceMetadataRequest): PullServiceMetadataRequest.AsObject;
  static serializeBinaryToWriter(message: PullServiceMetadataRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullServiceMetadataRequest;
  static deserializeBinaryFromReader(message: PullServiceMetadataRequest, reader: jspb.BinaryReader): PullServiceMetadataRequest;
}

export namespace PullServiceMetadataRequest {
  export type AsObject = {
    name: string;
    updatesOnly: boolean;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullServiceMetadataResponse extends jspb.Message {
  getChangesList(): Array<PullServiceMetadataResponse.Change>;
  setChangesList(value: Array<PullServiceMetadataResponse.Change>): PullServiceMetadataResponse;
  clearChangesList(): PullServiceMetadataResponse;
  addChanges(value?: PullServiceMetadataResponse.Change, index?: number): PullServiceMetadataResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullServiceMetadataResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullServiceMetadataResponse): PullServiceMetadataResponse.AsObject;
  static serializeBinaryToWriter(message: PullServiceMetadataResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullServiceMetadataResponse;
  static deserializeBinaryFromReader(message: PullServiceMetadataResponse, reader: jspb.BinaryReader): PullServiceMetadataResponse;
}

export namespace PullServiceMetadataResponse {
  export type AsObject = {
    changesList: Array<PullServiceMetadataResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getMetadata(): ServiceMetadata | undefined;
    setMetadata(value?: ServiceMetadata): Change;
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
      metadata?: ServiceMetadata.AsObject;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };
  }

}

