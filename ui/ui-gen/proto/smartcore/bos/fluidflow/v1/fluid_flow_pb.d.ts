import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '@smart-core-os/sc-api-grpc-web/types/info_pb'; // proto import: "types/info.proto"


export class FluidFlow extends jspb.Message {
  getTargetFlowRate(): number;
  setTargetFlowRate(value: number): FluidFlow;
  hasTargetFlowRate(): boolean;
  clearTargetFlowRate(): FluidFlow;

  getTargetDriveFrequency(): number;
  setTargetDriveFrequency(value: number): FluidFlow;
  hasTargetDriveFrequency(): boolean;
  clearTargetDriveFrequency(): FluidFlow;

  getFlowRate(): number;
  setFlowRate(value: number): FluidFlow;
  hasFlowRate(): boolean;
  clearFlowRate(): FluidFlow;

  getDriveFrequency(): number;
  setDriveFrequency(value: number): FluidFlow;
  hasDriveFrequency(): boolean;
  clearDriveFrequency(): FluidFlow;

  getDirection(): FluidFlow.Direction;
  setDirection(value: FluidFlow.Direction): FluidFlow;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): FluidFlow.AsObject;
  static toObject(includeInstance: boolean, msg: FluidFlow): FluidFlow.AsObject;
  static serializeBinaryToWriter(message: FluidFlow, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): FluidFlow;
  static deserializeBinaryFromReader(message: FluidFlow, reader: jspb.BinaryReader): FluidFlow;
}

export namespace FluidFlow {
  export type AsObject = {
    targetFlowRate?: number;
    targetDriveFrequency?: number;
    flowRate?: number;
    driveFrequency?: number;
    direction: FluidFlow.Direction;
  };

  export enum Direction {
    DIRECTION_UNSPECIFIED = 0,
    FLOW = 1,
    RETURN = 2,
    BLOCKING = 3,
  }

  export enum TargetFlowRateCase {
    _TARGET_FLOW_RATE_NOT_SET = 0,
    TARGET_FLOW_RATE = 1,
  }

  export enum TargetDriveFrequencyCase {
    _TARGET_DRIVE_FREQUENCY_NOT_SET = 0,
    TARGET_DRIVE_FREQUENCY = 2,
  }

  export enum FlowRateCase {
    _FLOW_RATE_NOT_SET = 0,
    FLOW_RATE = 3,
  }

  export enum DriveFrequencyCase {
    _DRIVE_FREQUENCY_NOT_SET = 0,
    DRIVE_FREQUENCY = 4,
  }
}

export class GetFluidFlowRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetFluidFlowRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetFluidFlowRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetFluidFlowRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetFluidFlowRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetFluidFlowRequest): GetFluidFlowRequest.AsObject;
  static serializeBinaryToWriter(message: GetFluidFlowRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetFluidFlowRequest;
  static deserializeBinaryFromReader(message: GetFluidFlowRequest, reader: jspb.BinaryReader): GetFluidFlowRequest;
}

export namespace GetFluidFlowRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class PullFluidFlowRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullFluidFlowRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullFluidFlowRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullFluidFlowRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullFluidFlowRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullFluidFlowRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullFluidFlowRequest): PullFluidFlowRequest.AsObject;
  static serializeBinaryToWriter(message: PullFluidFlowRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullFluidFlowRequest;
  static deserializeBinaryFromReader(message: PullFluidFlowRequest, reader: jspb.BinaryReader): PullFluidFlowRequest;
}

export namespace PullFluidFlowRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullFluidFlowResponse extends jspb.Message {
  getChangesList(): Array<PullFluidFlowResponse.Change>;
  setChangesList(value: Array<PullFluidFlowResponse.Change>): PullFluidFlowResponse;
  clearChangesList(): PullFluidFlowResponse;
  addChanges(value?: PullFluidFlowResponse.Change, index?: number): PullFluidFlowResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullFluidFlowResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullFluidFlowResponse): PullFluidFlowResponse.AsObject;
  static serializeBinaryToWriter(message: PullFluidFlowResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullFluidFlowResponse;
  static deserializeBinaryFromReader(message: PullFluidFlowResponse, reader: jspb.BinaryReader): PullFluidFlowResponse;
}

export namespace PullFluidFlowResponse {
  export type AsObject = {
    changesList: Array<PullFluidFlowResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getFlow(): FluidFlow | undefined;
    setFlow(value?: FluidFlow): Change;
    hasFlow(): boolean;
    clearFlow(): Change;

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
      flow?: FluidFlow.AsObject;
    };
  }

}

export class UpdateFluidFlowRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateFluidFlowRequest;

  getFlow(): FluidFlow | undefined;
  setFlow(value?: FluidFlow): UpdateFluidFlowRequest;
  hasFlow(): boolean;
  clearFlow(): UpdateFluidFlowRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateFluidFlowRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateFluidFlowRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateFluidFlowRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateFluidFlowRequest): UpdateFluidFlowRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateFluidFlowRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateFluidFlowRequest;
  static deserializeBinaryFromReader(message: UpdateFluidFlowRequest, reader: jspb.BinaryReader): UpdateFluidFlowRequest;
}

export namespace UpdateFluidFlowRequest {
  export type AsObject = {
    name: string;
    flow?: FluidFlow.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateFluidFlowResponse extends jspb.Message {
  getFlow(): FluidFlow | undefined;
  setFlow(value?: FluidFlow): UpdateFluidFlowResponse;
  hasFlow(): boolean;
  clearFlow(): UpdateFluidFlowResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateFluidFlowResponse.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateFluidFlowResponse): UpdateFluidFlowResponse.AsObject;
  static serializeBinaryToWriter(message: UpdateFluidFlowResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateFluidFlowResponse;
  static deserializeBinaryFromReader(message: UpdateFluidFlowResponse, reader: jspb.BinaryReader): UpdateFluidFlowResponse;
}

export namespace UpdateFluidFlowResponse {
  export type AsObject = {
    flow?: FluidFlow.AsObject;
  };
}

export class DescribeFluidFlowRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeFluidFlowRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeFluidFlowRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeFluidFlowRequest): DescribeFluidFlowRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeFluidFlowRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeFluidFlowRequest;
  static deserializeBinaryFromReader(message: DescribeFluidFlowRequest, reader: jspb.BinaryReader): DescribeFluidFlowRequest;
}

export namespace DescribeFluidFlowRequest {
  export type AsObject = {
    name: string;
  };
}

export class FluidFlowSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): FluidFlowSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): FluidFlowSupport;

  getFlowRateUnit(): string;
  setFlowRateUnit(value: string): FluidFlowSupport;

  getDriveFrequencyUnit(): string;
  setDriveFrequencyUnit(value: string): FluidFlowSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): FluidFlowSupport.AsObject;
  static toObject(includeInstance: boolean, msg: FluidFlowSupport): FluidFlowSupport.AsObject;
  static serializeBinaryToWriter(message: FluidFlowSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): FluidFlowSupport;
  static deserializeBinaryFromReader(message: FluidFlowSupport, reader: jspb.BinaryReader): FluidFlowSupport;
}

export namespace FluidFlowSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    flowRateUnit: string;
    driveFrequencyUnit: string;
  };
}

