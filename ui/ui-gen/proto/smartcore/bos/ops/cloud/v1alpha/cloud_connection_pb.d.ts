import * as jspb from 'google-protobuf'

import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as smartcore_bos_types_v1_change_pb from '../../../../../smartcore/bos/types/v1/change_pb'; // proto import: "smartcore/bos/types/v1/change.proto"


export class CloudConnection extends jspb.Message {
  getName(): string;
  setName(value: string): CloudConnection;

  getState(): CloudConnection.State;
  setState(value: CloudConnection.State): CloudConnection;

  getClientId(): string;
  setClientId(value: string): CloudConnection;

  getBosapiRoot(): string;
  setBosapiRoot(value: string): CloudConnection;

  getLastCheckInTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastCheckInTime(value?: google_protobuf_timestamp_pb.Timestamp): CloudConnection;
  hasLastCheckInTime(): boolean;
  clearLastCheckInTime(): CloudConnection;

  getLastError(): string;
  setLastError(value: string): CloudConnection;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CloudConnection.AsObject;
  static toObject(includeInstance: boolean, msg: CloudConnection): CloudConnection.AsObject;
  static serializeBinaryToWriter(message: CloudConnection, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CloudConnection;
  static deserializeBinaryFromReader(message: CloudConnection, reader: jspb.BinaryReader): CloudConnection;
}

export namespace CloudConnection {
  export type AsObject = {
    name: string;
    state: CloudConnection.State;
    clientId: string;
    bosapiRoot: string;
    lastCheckInTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    lastError: string;
  };

  export enum State {
    STATE_UNSPECIFIED = 0,
    UNCONFIGURED = 1,
    CONNECTING = 2,
    CONNECTED = 3,
    FAILED = 4,
  }
}

export class GetCloudConnectionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetCloudConnectionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetCloudConnectionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetCloudConnectionRequest): GetCloudConnectionRequest.AsObject;
  static serializeBinaryToWriter(message: GetCloudConnectionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetCloudConnectionRequest;
  static deserializeBinaryFromReader(message: GetCloudConnectionRequest, reader: jspb.BinaryReader): GetCloudConnectionRequest;
}

export namespace GetCloudConnectionRequest {
  export type AsObject = {
    name: string;
  };
}

export class GetCloudConnectionResponse extends jspb.Message {
  getCloudConnection(): CloudConnection | undefined;
  setCloudConnection(value?: CloudConnection): GetCloudConnectionResponse;
  hasCloudConnection(): boolean;
  clearCloudConnection(): GetCloudConnectionResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetCloudConnectionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetCloudConnectionResponse): GetCloudConnectionResponse.AsObject;
  static serializeBinaryToWriter(message: GetCloudConnectionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetCloudConnectionResponse;
  static deserializeBinaryFromReader(message: GetCloudConnectionResponse, reader: jspb.BinaryReader): GetCloudConnectionResponse;
}

export namespace GetCloudConnectionResponse {
  export type AsObject = {
    cloudConnection?: CloudConnection.AsObject;
  };
}

export class PullCloudConnectionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullCloudConnectionRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullCloudConnectionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullCloudConnectionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullCloudConnectionRequest): PullCloudConnectionRequest.AsObject;
  static serializeBinaryToWriter(message: PullCloudConnectionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullCloudConnectionRequest;
  static deserializeBinaryFromReader(message: PullCloudConnectionRequest, reader: jspb.BinaryReader): PullCloudConnectionRequest;
}

export namespace PullCloudConnectionRequest {
  export type AsObject = {
    name: string;
    updatesOnly: boolean;
  };
}

export class PullCloudConnectionResponse extends jspb.Message {
  getChangesList(): Array<PullCloudConnectionResponse.Change>;
  setChangesList(value: Array<PullCloudConnectionResponse.Change>): PullCloudConnectionResponse;
  clearChangesList(): PullCloudConnectionResponse;
  addChanges(value?: PullCloudConnectionResponse.Change, index?: number): PullCloudConnectionResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullCloudConnectionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullCloudConnectionResponse): PullCloudConnectionResponse.AsObject;
  static serializeBinaryToWriter(message: PullCloudConnectionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullCloudConnectionResponse;
  static deserializeBinaryFromReader(message: PullCloudConnectionResponse, reader: jspb.BinaryReader): PullCloudConnectionResponse;
}

export namespace PullCloudConnectionResponse {
  export type AsObject = {
    changesList: Array<PullCloudConnectionResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getType(): smartcore_bos_types_v1_change_pb.ChangeType;
    setType(value: smartcore_bos_types_v1_change_pb.ChangeType): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getCloudConnection(): CloudConnection | undefined;
    setCloudConnection(value?: CloudConnection): Change;
    hasCloudConnection(): boolean;
    clearCloudConnection(): Change;

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
      type: smartcore_bos_types_v1_change_pb.ChangeType;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      cloudConnection?: CloudConnection.AsObject;
    };
  }

}

export class RegisterCloudConnectionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): RegisterCloudConnectionRequest;

  getEnrollmentCode(): RegisterCloudConnectionRequest.EnrollmentCode | undefined;
  setEnrollmentCode(value?: RegisterCloudConnectionRequest.EnrollmentCode): RegisterCloudConnectionRequest;
  hasEnrollmentCode(): boolean;
  clearEnrollmentCode(): RegisterCloudConnectionRequest;

  getManual(): RegisterCloudConnectionRequest.ManualCredentials | undefined;
  setManual(value?: RegisterCloudConnectionRequest.ManualCredentials): RegisterCloudConnectionRequest;
  hasManual(): boolean;
  clearManual(): RegisterCloudConnectionRequest;

  getMethodCase(): RegisterCloudConnectionRequest.MethodCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RegisterCloudConnectionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RegisterCloudConnectionRequest): RegisterCloudConnectionRequest.AsObject;
  static serializeBinaryToWriter(message: RegisterCloudConnectionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RegisterCloudConnectionRequest;
  static deserializeBinaryFromReader(message: RegisterCloudConnectionRequest, reader: jspb.BinaryReader): RegisterCloudConnectionRequest;
}

export namespace RegisterCloudConnectionRequest {
  export type AsObject = {
    name: string;
    enrollmentCode?: RegisterCloudConnectionRequest.EnrollmentCode.AsObject;
    manual?: RegisterCloudConnectionRequest.ManualCredentials.AsObject;
  };

  export class EnrollmentCode extends jspb.Message {
    getCode(): string;
    setCode(value: string): EnrollmentCode;

    getRegisterUrl(): string;
    setRegisterUrl(value: string): EnrollmentCode;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): EnrollmentCode.AsObject;
    static toObject(includeInstance: boolean, msg: EnrollmentCode): EnrollmentCode.AsObject;
    static serializeBinaryToWriter(message: EnrollmentCode, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): EnrollmentCode;
    static deserializeBinaryFromReader(message: EnrollmentCode, reader: jspb.BinaryReader): EnrollmentCode;
  }

  export namespace EnrollmentCode {
    export type AsObject = {
      code: string;
      registerUrl: string;
    };
  }


  export class ManualCredentials extends jspb.Message {
    getClientId(): string;
    setClientId(value: string): ManualCredentials;

    getClientSecret(): string;
    setClientSecret(value: string): ManualCredentials;

    getBosapiRoot(): string;
    setBosapiRoot(value: string): ManualCredentials;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ManualCredentials.AsObject;
    static toObject(includeInstance: boolean, msg: ManualCredentials): ManualCredentials.AsObject;
    static serializeBinaryToWriter(message: ManualCredentials, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ManualCredentials;
    static deserializeBinaryFromReader(message: ManualCredentials, reader: jspb.BinaryReader): ManualCredentials;
  }

  export namespace ManualCredentials {
    export type AsObject = {
      clientId: string;
      clientSecret: string;
      bosapiRoot: string;
    };
  }


  export enum MethodCase {
    METHOD_NOT_SET = 0,
    ENROLLMENT_CODE = 2,
    MANUAL = 3,
  }
}

export class RegisterCloudConnectionResponse extends jspb.Message {
  getCloudConnection(): CloudConnection | undefined;
  setCloudConnection(value?: CloudConnection): RegisterCloudConnectionResponse;
  hasCloudConnection(): boolean;
  clearCloudConnection(): RegisterCloudConnectionResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RegisterCloudConnectionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: RegisterCloudConnectionResponse): RegisterCloudConnectionResponse.AsObject;
  static serializeBinaryToWriter(message: RegisterCloudConnectionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RegisterCloudConnectionResponse;
  static deserializeBinaryFromReader(message: RegisterCloudConnectionResponse, reader: jspb.BinaryReader): RegisterCloudConnectionResponse;
}

export namespace RegisterCloudConnectionResponse {
  export type AsObject = {
    cloudConnection?: CloudConnection.AsObject;
  };
}

export class UnlinkCloudConnectionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UnlinkCloudConnectionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UnlinkCloudConnectionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UnlinkCloudConnectionRequest): UnlinkCloudConnectionRequest.AsObject;
  static serializeBinaryToWriter(message: UnlinkCloudConnectionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UnlinkCloudConnectionRequest;
  static deserializeBinaryFromReader(message: UnlinkCloudConnectionRequest, reader: jspb.BinaryReader): UnlinkCloudConnectionRequest;
}

export namespace UnlinkCloudConnectionRequest {
  export type AsObject = {
    name: string;
  };
}

export class UnlinkCloudConnectionResponse extends jspb.Message {
  getCloudConnection(): CloudConnection | undefined;
  setCloudConnection(value?: CloudConnection): UnlinkCloudConnectionResponse;
  hasCloudConnection(): boolean;
  clearCloudConnection(): UnlinkCloudConnectionResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UnlinkCloudConnectionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: UnlinkCloudConnectionResponse): UnlinkCloudConnectionResponse.AsObject;
  static serializeBinaryToWriter(message: UnlinkCloudConnectionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UnlinkCloudConnectionResponse;
  static deserializeBinaryFromReader(message: UnlinkCloudConnectionResponse, reader: jspb.BinaryReader): UnlinkCloudConnectionResponse;
}

export namespace UnlinkCloudConnectionResponse {
  export type AsObject = {
    cloudConnection?: CloudConnection.AsObject;
  };
}

export class GetCloudConnectionDefaultsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetCloudConnectionDefaultsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetCloudConnectionDefaultsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetCloudConnectionDefaultsRequest): GetCloudConnectionDefaultsRequest.AsObject;
  static serializeBinaryToWriter(message: GetCloudConnectionDefaultsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetCloudConnectionDefaultsRequest;
  static deserializeBinaryFromReader(message: GetCloudConnectionDefaultsRequest, reader: jspb.BinaryReader): GetCloudConnectionDefaultsRequest;
}

export namespace GetCloudConnectionDefaultsRequest {
  export type AsObject = {
    name: string;
  };
}

export class GetCloudConnectionDefaultsResponse extends jspb.Message {
  getDefaults(): CloudConnectionDefaults | undefined;
  setDefaults(value?: CloudConnectionDefaults): GetCloudConnectionDefaultsResponse;
  hasDefaults(): boolean;
  clearDefaults(): GetCloudConnectionDefaultsResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetCloudConnectionDefaultsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GetCloudConnectionDefaultsResponse): GetCloudConnectionDefaultsResponse.AsObject;
  static serializeBinaryToWriter(message: GetCloudConnectionDefaultsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetCloudConnectionDefaultsResponse;
  static deserializeBinaryFromReader(message: GetCloudConnectionDefaultsResponse, reader: jspb.BinaryReader): GetCloudConnectionDefaultsResponse;
}

export namespace GetCloudConnectionDefaultsResponse {
  export type AsObject = {
    defaults?: CloudConnectionDefaults.AsObject;
  };
}

export class CloudConnectionDefaults extends jspb.Message {
  getRegisterUrl(): string;
  setRegisterUrl(value: string): CloudConnectionDefaults;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CloudConnectionDefaults.AsObject;
  static toObject(includeInstance: boolean, msg: CloudConnectionDefaults): CloudConnectionDefaults.AsObject;
  static serializeBinaryToWriter(message: CloudConnectionDefaults, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CloudConnectionDefaults;
  static deserializeBinaryFromReader(message: CloudConnectionDefaults, reader: jspb.BinaryReader): CloudConnectionDefaults;
}

export namespace CloudConnectionDefaults {
  export type AsObject = {
    registerUrl: string;
  };
}

export class TestCloudConnectionRequest extends jspb.Message {
  getName(): string;
  setName(value: string): TestCloudConnectionRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TestCloudConnectionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: TestCloudConnectionRequest): TestCloudConnectionRequest.AsObject;
  static serializeBinaryToWriter(message: TestCloudConnectionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TestCloudConnectionRequest;
  static deserializeBinaryFromReader(message: TestCloudConnectionRequest, reader: jspb.BinaryReader): TestCloudConnectionRequest;
}

export namespace TestCloudConnectionRequest {
  export type AsObject = {
    name: string;
  };
}

export class TestCloudConnectionResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TestCloudConnectionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TestCloudConnectionResponse): TestCloudConnectionResponse.AsObject;
  static serializeBinaryToWriter(message: TestCloudConnectionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TestCloudConnectionResponse;
  static deserializeBinaryFromReader(message: TestCloudConnectionResponse, reader: jspb.BinaryReader): TestCloudConnectionResponse;
}

export namespace TestCloudConnectionResponse {
  export type AsObject = {
  };
}

