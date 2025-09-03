import * as jspb from 'google-protobuf'

import * as google_protobuf_empty_pb from 'google-protobuf/google/protobuf/empty_pb'; // proto import: "google/protobuf/empty.proto"
import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as traits_metadata_pb from '@smart-core-os/sc-api-grpc-web/traits/metadata_pb'; // proto import: "traits/metadata.proto"
import * as types_change_pb from '@smart-core-os/sc-api-grpc-web/types/change_pb'; // proto import: "types/change.proto"
import * as types_time_period_pb from '@smart-core-os/sc-api-grpc-web/types/time/period_pb'; // proto import: "types/time/period.proto"
import * as health_pb from './health_pb'; // proto import: "health.proto"


export class Device extends jspb.Message {
  getName(): string;
  setName(value: string): Device;

  getMetadata(): traits_metadata_pb.Metadata | undefined;
  setMetadata(value?: traits_metadata_pb.Metadata): Device;
  hasMetadata(): boolean;
  clearMetadata(): Device;

  getHealthChecksList(): Array<health_pb.HealthCheck>;
  setHealthChecksList(value: Array<health_pb.HealthCheck>): Device;
  clearHealthChecksList(): Device;
  addHealthChecks(value?: health_pb.HealthCheck, index?: number): health_pb.HealthCheck;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Device.AsObject;
  static toObject(includeInstance: boolean, msg: Device): Device.AsObject;
  static serializeBinaryToWriter(message: Device, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Device;
  static deserializeBinaryFromReader(message: Device, reader: jspb.BinaryReader): Device;
}

export namespace Device {
  export type AsObject = {
    name: string,
    metadata?: traits_metadata_pb.Metadata.AsObject,
    healthChecksList: Array<health_pb.HealthCheck.AsObject>,
  }

  export class Query extends jspb.Message {
    getConditionsList(): Array<Device.Query.Condition>;
    setConditionsList(value: Array<Device.Query.Condition>): Query;
    clearConditionsList(): Query;
    addConditions(value?: Device.Query.Condition, index?: number): Device.Query.Condition;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Query.AsObject;
    static toObject(includeInstance: boolean, msg: Query): Query.AsObject;
    static serializeBinaryToWriter(message: Query, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Query;
    static deserializeBinaryFromReader(message: Query, reader: jspb.BinaryReader): Query;
  }

  export namespace Query {
    export type AsObject = {
      conditionsList: Array<Device.Query.Condition.AsObject>,
    }

    export class Condition extends jspb.Message {
      getField(): string;
      setField(value: string): Condition;

      getStringEqual(): string;
      setStringEqual(value: string): Condition;

      getStringEqualFold(): string;
      setStringEqualFold(value: string): Condition;

      getStringContains(): string;
      setStringContains(value: string): Condition;

      getStringContainsFold(): string;
      setStringContainsFold(value: string): Condition;

      getStringIn(): Device.Query.StringList | undefined;
      setStringIn(value?: Device.Query.StringList): Condition;
      hasStringIn(): boolean;
      clearStringIn(): Condition;

      getStringInFold(): Device.Query.StringList | undefined;
      setStringInFold(value?: Device.Query.StringList): Condition;
      hasStringInFold(): boolean;
      clearStringInFold(): Condition;

      getTimestampEqual(): google_protobuf_timestamp_pb.Timestamp | undefined;
      setTimestampEqual(value?: google_protobuf_timestamp_pb.Timestamp): Condition;
      hasTimestampEqual(): boolean;
      clearTimestampEqual(): Condition;

      getTimestampGt(): google_protobuf_timestamp_pb.Timestamp | undefined;
      setTimestampGt(value?: google_protobuf_timestamp_pb.Timestamp): Condition;
      hasTimestampGt(): boolean;
      clearTimestampGt(): Condition;

      getTimestampGte(): google_protobuf_timestamp_pb.Timestamp | undefined;
      setTimestampGte(value?: google_protobuf_timestamp_pb.Timestamp): Condition;
      hasTimestampGte(): boolean;
      clearTimestampGte(): Condition;

      getTimestampLt(): google_protobuf_timestamp_pb.Timestamp | undefined;
      setTimestampLt(value?: google_protobuf_timestamp_pb.Timestamp): Condition;
      hasTimestampLt(): boolean;
      clearTimestampLt(): Condition;

      getTimestampLte(): google_protobuf_timestamp_pb.Timestamp | undefined;
      setTimestampLte(value?: google_protobuf_timestamp_pb.Timestamp): Condition;
      hasTimestampLte(): boolean;
      clearTimestampLte(): Condition;

      getNameDescendant(): string;
      setNameDescendant(value: string): Condition;

      getNameDescendantInc(): string;
      setNameDescendantInc(value: string): Condition;

      getNameDescendantIn(): Device.Query.StringList | undefined;
      setNameDescendantIn(value?: Device.Query.StringList): Condition;
      hasNameDescendantIn(): boolean;
      clearNameDescendantIn(): Condition;

      getNameDescendantIncIn(): Device.Query.StringList | undefined;
      setNameDescendantIncIn(value?: Device.Query.StringList): Condition;
      hasNameDescendantIncIn(): boolean;
      clearNameDescendantIncIn(): Condition;

      getPresent(): google_protobuf_empty_pb.Empty | undefined;
      setPresent(value?: google_protobuf_empty_pb.Empty): Condition;
      hasPresent(): boolean;
      clearPresent(): Condition;

      getMatches(): Device.Query | undefined;
      setMatches(value?: Device.Query): Condition;
      hasMatches(): boolean;
      clearMatches(): Condition;

      getValueCase(): Condition.ValueCase;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): Condition.AsObject;
      static toObject(includeInstance: boolean, msg: Condition): Condition.AsObject;
      static serializeBinaryToWriter(message: Condition, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): Condition;
      static deserializeBinaryFromReader(message: Condition, reader: jspb.BinaryReader): Condition;
    }

    export namespace Condition {
      export type AsObject = {
        field: string,
        stringEqual: string,
        stringEqualFold: string,
        stringContains: string,
        stringContainsFold: string,
        stringIn?: Device.Query.StringList.AsObject,
        stringInFold?: Device.Query.StringList.AsObject,
        timestampEqual?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        timestampGt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        timestampGte?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        timestampLt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        timestampLte?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        nameDescendant: string,
        nameDescendantInc: string,
        nameDescendantIn?: Device.Query.StringList.AsObject,
        nameDescendantIncIn?: Device.Query.StringList.AsObject,
        present?: google_protobuf_empty_pb.Empty.AsObject,
        matches?: Device.Query.AsObject,
      }

      export enum ValueCase { 
        VALUE_NOT_SET = 0,
        STRING_EQUAL = 2,
        STRING_EQUAL_FOLD = 3,
        STRING_CONTAINS = 4,
        STRING_CONTAINS_FOLD = 5,
        STRING_IN = 6,
        STRING_IN_FOLD = 7,
        TIMESTAMP_EQUAL = 20,
        TIMESTAMP_GT = 21,
        TIMESTAMP_GTE = 22,
        TIMESTAMP_LT = 23,
        TIMESTAMP_LTE = 24,
        NAME_DESCENDANT = 30,
        NAME_DESCENDANT_INC = 31,
        NAME_DESCENDANT_IN = 32,
        NAME_DESCENDANT_INC_IN = 33,
        PRESENT = 40,
        MATCHES = 50,
      }
    }


    export class StringList extends jspb.Message {
      getStringsList(): Array<string>;
      setStringsList(value: Array<string>): StringList;
      clearStringsList(): StringList;
      addStrings(value: string, index?: number): StringList;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): StringList.AsObject;
      static toObject(includeInstance: boolean, msg: StringList): StringList.AsObject;
      static serializeBinaryToWriter(message: StringList, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): StringList;
      static deserializeBinaryFromReader(message: StringList, reader: jspb.BinaryReader): StringList;
    }

    export namespace StringList {
      export type AsObject = {
        stringsList: Array<string>,
      }
    }

  }

}

export class DevicesMetadata extends jspb.Message {
  getTotalCount(): number;
  setTotalCount(value: number): DevicesMetadata;

  getFieldCountsList(): Array<DevicesMetadata.StringFieldCount>;
  setFieldCountsList(value: Array<DevicesMetadata.StringFieldCount>): DevicesMetadata;
  clearFieldCountsList(): DevicesMetadata;
  addFieldCounts(value?: DevicesMetadata.StringFieldCount, index?: number): DevicesMetadata.StringFieldCount;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DevicesMetadata.AsObject;
  static toObject(includeInstance: boolean, msg: DevicesMetadata): DevicesMetadata.AsObject;
  static serializeBinaryToWriter(message: DevicesMetadata, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DevicesMetadata;
  static deserializeBinaryFromReader(message: DevicesMetadata, reader: jspb.BinaryReader): DevicesMetadata;
}

export namespace DevicesMetadata {
  export type AsObject = {
    totalCount: number,
    fieldCountsList: Array<DevicesMetadata.StringFieldCount.AsObject>,
  }

  export class StringFieldCount extends jspb.Message {
    getField(): string;
    setField(value: string): StringFieldCount;

    getCountsMap(): jspb.Map<string, number>;
    clearCountsMap(): StringFieldCount;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): StringFieldCount.AsObject;
    static toObject(includeInstance: boolean, msg: StringFieldCount): StringFieldCount.AsObject;
    static serializeBinaryToWriter(message: StringFieldCount, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): StringFieldCount;
    static deserializeBinaryFromReader(message: StringFieldCount, reader: jspb.BinaryReader): StringFieldCount;
  }

  export namespace StringFieldCount {
    export type AsObject = {
      field: string,
      countsMap: Array<[string, number]>,
    }
  }


  export class Include extends jspb.Message {
    getFieldsList(): Array<string>;
    setFieldsList(value: Array<string>): Include;
    clearFieldsList(): Include;
    addFields(value: string, index?: number): Include;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Include.AsObject;
    static toObject(includeInstance: boolean, msg: Include): Include.AsObject;
    static serializeBinaryToWriter(message: Include, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Include;
    static deserializeBinaryFromReader(message: Include, reader: jspb.BinaryReader): Include;
  }

  export namespace Include {
    export type AsObject = {
      fieldsList: Array<string>,
    }
  }

}

export class ListDevicesRequest extends jspb.Message {
  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListDevicesRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListDevicesRequest;

  getPageSize(): number;
  setPageSize(value: number): ListDevicesRequest;

  getPageToken(): string;
  setPageToken(value: string): ListDevicesRequest;

  getQuery(): Device.Query | undefined;
  setQuery(value?: Device.Query): ListDevicesRequest;
  hasQuery(): boolean;
  clearQuery(): ListDevicesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListDevicesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListDevicesRequest): ListDevicesRequest.AsObject;
  static serializeBinaryToWriter(message: ListDevicesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListDevicesRequest;
  static deserializeBinaryFromReader(message: ListDevicesRequest, reader: jspb.BinaryReader): ListDevicesRequest;
}

export namespace ListDevicesRequest {
  export type AsObject = {
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject,
    pageSize: number,
    pageToken: string,
    query?: Device.Query.AsObject,
  }
}

export class ListDevicesResponse extends jspb.Message {
  getDevicesList(): Array<Device>;
  setDevicesList(value: Array<Device>): ListDevicesResponse;
  clearDevicesList(): ListDevicesResponse;
  addDevices(value?: Device, index?: number): Device;

  getNextPageToken(): string;
  setNextPageToken(value: string): ListDevicesResponse;

  getTotalSize(): number;
  setTotalSize(value: number): ListDevicesResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListDevicesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListDevicesResponse): ListDevicesResponse.AsObject;
  static serializeBinaryToWriter(message: ListDevicesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListDevicesResponse;
  static deserializeBinaryFromReader(message: ListDevicesResponse, reader: jspb.BinaryReader): ListDevicesResponse;
}

export namespace ListDevicesResponse {
  export type AsObject = {
    devicesList: Array<Device.AsObject>,
    nextPageToken: string,
    totalSize: number,
  }
}

export class PullDevicesRequest extends jspb.Message {
  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullDevicesRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullDevicesRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullDevicesRequest;

  getQuery(): Device.Query | undefined;
  setQuery(value?: Device.Query): PullDevicesRequest;
  hasQuery(): boolean;
  clearQuery(): PullDevicesRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullDevicesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullDevicesRequest): PullDevicesRequest.AsObject;
  static serializeBinaryToWriter(message: PullDevicesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullDevicesRequest;
  static deserializeBinaryFromReader(message: PullDevicesRequest, reader: jspb.BinaryReader): PullDevicesRequest;
}

export namespace PullDevicesRequest {
  export type AsObject = {
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject,
    updatesOnly: boolean,
    query?: Device.Query.AsObject,
  }
}

export class PullDevicesResponse extends jspb.Message {
  getChangesList(): Array<PullDevicesResponse.Change>;
  setChangesList(value: Array<PullDevicesResponse.Change>): PullDevicesResponse;
  clearChangesList(): PullDevicesResponse;
  addChanges(value?: PullDevicesResponse.Change, index?: number): PullDevicesResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullDevicesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullDevicesResponse): PullDevicesResponse.AsObject;
  static serializeBinaryToWriter(message: PullDevicesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullDevicesResponse;
  static deserializeBinaryFromReader(message: PullDevicesResponse, reader: jspb.BinaryReader): PullDevicesResponse;
}

export namespace PullDevicesResponse {
  export type AsObject = {
    changesList: Array<PullDevicesResponse.Change.AsObject>,
  }

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getType(): types_change_pb.ChangeType;
    setType(value: types_change_pb.ChangeType): Change;

    getNewValue(): Device | undefined;
    setNewValue(value?: Device): Change;
    hasNewValue(): boolean;
    clearNewValue(): Change;

    getOldValue(): Device | undefined;
    setOldValue(value?: Device): Change;
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
      name: string,
      type: types_change_pb.ChangeType,
      newValue?: Device.AsObject,
      oldValue?: Device.AsObject,
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    }
  }

}

export class GetDevicesMetadataRequest extends jspb.Message {
  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetDevicesMetadataRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetDevicesMetadataRequest;

  getIncludes(): DevicesMetadata.Include | undefined;
  setIncludes(value?: DevicesMetadata.Include): GetDevicesMetadataRequest;
  hasIncludes(): boolean;
  clearIncludes(): GetDevicesMetadataRequest;

  getQuery(): Device.Query | undefined;
  setQuery(value?: Device.Query): GetDevicesMetadataRequest;
  hasQuery(): boolean;
  clearQuery(): GetDevicesMetadataRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDevicesMetadataRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetDevicesMetadataRequest): GetDevicesMetadataRequest.AsObject;
  static serializeBinaryToWriter(message: GetDevicesMetadataRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDevicesMetadataRequest;
  static deserializeBinaryFromReader(message: GetDevicesMetadataRequest, reader: jspb.BinaryReader): GetDevicesMetadataRequest;
}

export namespace GetDevicesMetadataRequest {
  export type AsObject = {
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject,
    includes?: DevicesMetadata.Include.AsObject,
    query?: Device.Query.AsObject,
  }
}

export class PullDevicesMetadataRequest extends jspb.Message {
  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullDevicesMetadataRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullDevicesMetadataRequest;

  getIncludes(): DevicesMetadata.Include | undefined;
  setIncludes(value?: DevicesMetadata.Include): PullDevicesMetadataRequest;
  hasIncludes(): boolean;
  clearIncludes(): PullDevicesMetadataRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullDevicesMetadataRequest;

  getQuery(): Device.Query | undefined;
  setQuery(value?: Device.Query): PullDevicesMetadataRequest;
  hasQuery(): boolean;
  clearQuery(): PullDevicesMetadataRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullDevicesMetadataRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullDevicesMetadataRequest): PullDevicesMetadataRequest.AsObject;
  static serializeBinaryToWriter(message: PullDevicesMetadataRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullDevicesMetadataRequest;
  static deserializeBinaryFromReader(message: PullDevicesMetadataRequest, reader: jspb.BinaryReader): PullDevicesMetadataRequest;
}

export namespace PullDevicesMetadataRequest {
  export type AsObject = {
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject,
    includes?: DevicesMetadata.Include.AsObject,
    updatesOnly: boolean,
    query?: Device.Query.AsObject,
  }
}

export class PullDevicesMetadataResponse extends jspb.Message {
  getChangesList(): Array<PullDevicesMetadataResponse.Change>;
  setChangesList(value: Array<PullDevicesMetadataResponse.Change>): PullDevicesMetadataResponse;
  clearChangesList(): PullDevicesMetadataResponse;
  addChanges(value?: PullDevicesMetadataResponse.Change, index?: number): PullDevicesMetadataResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullDevicesMetadataResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullDevicesMetadataResponse): PullDevicesMetadataResponse.AsObject;
  static serializeBinaryToWriter(message: PullDevicesMetadataResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullDevicesMetadataResponse;
  static deserializeBinaryFromReader(message: PullDevicesMetadataResponse, reader: jspb.BinaryReader): PullDevicesMetadataResponse;
}

export namespace PullDevicesMetadataResponse {
  export type AsObject = {
    changesList: Array<PullDevicesMetadataResponse.Change.AsObject>,
  }

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getDevicesMetadata(): DevicesMetadata | undefined;
    setDevicesMetadata(value?: DevicesMetadata): Change;
    hasDevicesMetadata(): boolean;
    clearDevicesMetadata(): Change;

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
      name: string,
      devicesMetadata?: DevicesMetadata.AsObject,
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    }
  }

}

export class GetDownloadDevicesUrlRequest extends jspb.Message {
  getQuery(): Device.Query | undefined;
  setQuery(value?: Device.Query): GetDownloadDevicesUrlRequest;
  hasQuery(): boolean;
  clearQuery(): GetDownloadDevicesUrlRequest;

  getMediaType(): string;
  setMediaType(value: string): GetDownloadDevicesUrlRequest;

  getHistory(): types_time_period_pb.Period | undefined;
  setHistory(value?: types_time_period_pb.Period): GetDownloadDevicesUrlRequest;
  hasHistory(): boolean;
  clearHistory(): GetDownloadDevicesUrlRequest;

  getTable(): GetDownloadDevicesUrlRequest.Table | undefined;
  setTable(value?: GetDownloadDevicesUrlRequest.Table): GetDownloadDevicesUrlRequest;
  hasTable(): boolean;
  clearTable(): GetDownloadDevicesUrlRequest;

  getFilename(): string;
  setFilename(value: string): GetDownloadDevicesUrlRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDownloadDevicesUrlRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetDownloadDevicesUrlRequest): GetDownloadDevicesUrlRequest.AsObject;
  static serializeBinaryToWriter(message: GetDownloadDevicesUrlRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDownloadDevicesUrlRequest;
  static deserializeBinaryFromReader(message: GetDownloadDevicesUrlRequest, reader: jspb.BinaryReader): GetDownloadDevicesUrlRequest;
}

export namespace GetDownloadDevicesUrlRequest {
  export type AsObject = {
    query?: Device.Query.AsObject,
    mediaType: string,
    history?: types_time_period_pb.Period.AsObject,
    table?: GetDownloadDevicesUrlRequest.Table.AsObject,
    filename: string,
  }

  export class Table extends jspb.Message {
    getIncludeColsList(): Array<GetDownloadDevicesUrlRequest.Table.Column>;
    setIncludeColsList(value: Array<GetDownloadDevicesUrlRequest.Table.Column>): Table;
    clearIncludeColsList(): Table;
    addIncludeCols(value?: GetDownloadDevicesUrlRequest.Table.Column, index?: number): GetDownloadDevicesUrlRequest.Table.Column;

    getExcludeColsList(): Array<GetDownloadDevicesUrlRequest.Table.Column>;
    setExcludeColsList(value: Array<GetDownloadDevicesUrlRequest.Table.Column>): Table;
    clearExcludeColsList(): Table;
    addExcludeCols(value?: GetDownloadDevicesUrlRequest.Table.Column, index?: number): GetDownloadDevicesUrlRequest.Table.Column;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Table.AsObject;
    static toObject(includeInstance: boolean, msg: Table): Table.AsObject;
    static serializeBinaryToWriter(message: Table, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Table;
    static deserializeBinaryFromReader(message: Table, reader: jspb.BinaryReader): Table;
  }

  export namespace Table {
    export type AsObject = {
      includeColsList: Array<GetDownloadDevicesUrlRequest.Table.Column.AsObject>,
      excludeColsList: Array<GetDownloadDevicesUrlRequest.Table.Column.AsObject>,
    }

    export class Column extends jspb.Message {
      getName(): string;
      setName(value: string): Column;

      getTitle(): string;
      setTitle(value: string): Column;

      serializeBinary(): Uint8Array;
      toObject(includeInstance?: boolean): Column.AsObject;
      static toObject(includeInstance: boolean, msg: Column): Column.AsObject;
      static serializeBinaryToWriter(message: Column, writer: jspb.BinaryWriter): void;
      static deserializeBinary(bytes: Uint8Array): Column;
      static deserializeBinaryFromReader(message: Column, reader: jspb.BinaryReader): Column;
    }

    export namespace Column {
      export type AsObject = {
        name: string,
        title: string,
      }
    }

  }

}

export class DownloadDevicesUrl extends jspb.Message {
  getUrl(): string;
  setUrl(value: string): DownloadDevicesUrl;

  getFilename(): string;
  setFilename(value: string): DownloadDevicesUrl;

  getMediaType(): string;
  setMediaType(value: string): DownloadDevicesUrl;

  getExpireAfterTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setExpireAfterTime(value?: google_protobuf_timestamp_pb.Timestamp): DownloadDevicesUrl;
  hasExpireAfterTime(): boolean;
  clearExpireAfterTime(): DownloadDevicesUrl;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DownloadDevicesUrl.AsObject;
  static toObject(includeInstance: boolean, msg: DownloadDevicesUrl): DownloadDevicesUrl.AsObject;
  static serializeBinaryToWriter(message: DownloadDevicesUrl, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DownloadDevicesUrl;
  static deserializeBinaryFromReader(message: DownloadDevicesUrl, reader: jspb.BinaryReader): DownloadDevicesUrl;
}

export namespace DownloadDevicesUrl {
  export type AsObject = {
    url: string,
    filename: string,
    mediaType: string,
    expireAfterTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

