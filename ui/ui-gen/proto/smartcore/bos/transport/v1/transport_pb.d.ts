import * as jspb from 'google-protobuf'

import * as google_protobuf_duration_pb from 'google-protobuf/google/protobuf/duration_pb'; // proto import: "google/protobuf/duration.proto"
import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_info_pb from '@smart-core-os/sc-api-grpc-web/types/info_pb'; // proto import: "types/info.proto"


export class Transport extends jspb.Message {
  getActualPosition(): Transport.Location | undefined;
  setActualPosition(value?: Transport.Location): Transport;
  hasActualPosition(): boolean;
  clearActualPosition(): Transport;

  getNextDestinationsList(): Array<Transport.Location>;
  setNextDestinationsList(value: Array<Transport.Location>): Transport;
  clearNextDestinationsList(): Transport;
  addNextDestinations(value?: Transport.Location, index?: number): Transport.Location;

  getMovingDirection(): Transport.Direction;
  setMovingDirection(value: Transport.Direction): Transport;

  getLoad(): number;
  setLoad(value: number): Transport;
  hasLoad(): boolean;
  clearLoad(): Transport;

  getDoorsList(): Array<Transport.Door>;
  setDoorsList(value: Array<Transport.Door>): Transport;
  clearDoorsList(): Transport;
  addDoors(value?: Transport.Door, index?: number): Transport.Door;

  getOperatingMode(): Transport.OperatingMode;
  setOperatingMode(value: Transport.OperatingMode): Transport;

  getFaultsList(): Array<Transport.Fault>;
  setFaultsList(value: Array<Transport.Fault>): Transport;
  clearFaultsList(): Transport;
  addFaults(value?: Transport.Fault, index?: number): Transport.Fault;

  getPassengerAlarm(): Transport.Alarm | undefined;
  setPassengerAlarm(value?: Transport.Alarm): Transport;
  hasPassengerAlarm(): boolean;
  clearPassengerAlarm(): Transport;

  getSpeed(): number;
  setSpeed(value: number): Transport;
  hasSpeed(): boolean;
  clearSpeed(): Transport;

  getSupportedDestinationsList(): Array<Transport.Location>;
  setSupportedDestinationsList(value: Array<Transport.Location>): Transport;
  clearSupportedDestinationsList(): Transport;
  addSupportedDestinations(value?: Transport.Location, index?: number): Transport.Location;

  getActive(): Transport.Active;
  setActive(value: Transport.Active): Transport;

  getPayloadsList(): Array<Transport.Payload>;
  setPayloadsList(value: Array<Transport.Payload>): Transport;
  clearPayloadsList(): Transport;
  addPayloads(value?: Transport.Payload, index?: number): Transport.Payload;

  getEtaToNextDestination(): google_protobuf_duration_pb.Duration | undefined;
  setEtaToNextDestination(value?: google_protobuf_duration_pb.Duration): Transport;
  hasEtaToNextDestination(): boolean;
  clearEtaToNextDestination(): Transport;

  getStoppedReason(): Transport.StoppedReason | undefined;
  setStoppedReason(value?: Transport.StoppedReason): Transport;
  hasStoppedReason(): boolean;
  clearStoppedReason(): Transport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Transport.AsObject;
  static toObject(includeInstance: boolean, msg: Transport): Transport.AsObject;
  static serializeBinaryToWriter(message: Transport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Transport;
  static deserializeBinaryFromReader(message: Transport, reader: jspb.BinaryReader): Transport;
}

export namespace Transport {
  export type AsObject = {
    actualPosition?: Transport.Location.AsObject;
    nextDestinationsList: Array<Transport.Location.AsObject>;
    movingDirection: Transport.Direction;
    load?: number;
    doorsList: Array<Transport.Door.AsObject>;
    operatingMode: Transport.OperatingMode;
    faultsList: Array<Transport.Fault.AsObject>;
    passengerAlarm?: Transport.Alarm.AsObject;
    speed?: number;
    supportedDestinationsList: Array<Transport.Location.AsObject>;
    active: Transport.Active;
    payloadsList: Array<Transport.Payload.AsObject>;
    etaToNextDestination?: google_protobuf_duration_pb.Duration.AsObject;
    stoppedReason?: Transport.StoppedReason.AsObject;
  };

  export class Alarm extends jspb.Message {
    getState(): Transport.Alarm.AlarmState;
    setState(value: Transport.Alarm.AlarmState): Alarm;

    getTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setTime(value?: google_protobuf_timestamp_pb.Timestamp): Alarm;
    hasTime(): boolean;
    clearTime(): Alarm;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Alarm.AsObject;
    static toObject(includeInstance: boolean, msg: Alarm): Alarm.AsObject;
    static serializeBinaryToWriter(message: Alarm, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Alarm;
    static deserializeBinaryFromReader(message: Alarm, reader: jspb.BinaryReader): Alarm;
  }

  export namespace Alarm {
    export type AsObject = {
      state: Transport.Alarm.AlarmState;
      time?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };

    export enum AlarmState {
      ALARM_STATE_UNSPECIFIED = 0,
      UNACTIVATED = 1,
      ACTIVATED = 2,
    }
  }


  export class Fault extends jspb.Message {
    getFaultType(): Transport.Fault.FaultType;
    setFaultType(value: Transport.Fault.FaultType): Fault;

    getTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setTime(value?: google_protobuf_timestamp_pb.Timestamp): Fault;
    hasTime(): boolean;
    clearTime(): Fault;

    getDescription(): string;
    setDescription(value: string): Fault;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Fault.AsObject;
    static toObject(includeInstance: boolean, msg: Fault): Fault.AsObject;
    static serializeBinaryToWriter(message: Fault, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Fault;
    static deserializeBinaryFromReader(message: Fault, reader: jspb.BinaryReader): Fault;
  }

  export namespace Fault {
    export type AsObject = {
      faultType: Transport.Fault.FaultType;
      time?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      description: string;
    };

    export enum FaultType {
      FAULT_TYPE_UNSPECIFIED = 0,
      CONTROLLER_FAULT = 1,
      DRIVE_AND_MOTOR_FAULT = 2,
      MECHANICAL_COMPONENT_FAULT = 3,
      OVERSPEED_FAULT = 4,
      POWER_SUPPLY_FAULT = 5,
      SAFETY_DEVICE_FAULT = 6,
      CONTROLLER_SUPPLY_FAULT = 7,
      DRIVE_TEMPERATURE_EXCEEDED = 8,
      COMB_PLATE_FAULT = 9,
      GENERAL_FAULT = 10,
      DOOR_FAULT = 11,
      LEVELLING_FAULT = 12,
      SAFETY_CIRCUIT_BREAK_FAULT = 13,
      FAIL_TO_START = 14,
      ALARM_BUTTON = 15,
      DOOR_NOT_CLOSING = 16,
      DOOR_NOT_OPENING = 17,
      GOVERNOR_AND_SAFETY_GEAR_FAULT = 18,
      LIFT_SHAFT_DEVICE_FAULT = 19,
      CAR_STOPPED_OUTSIDE_LANDING_ZONE = 20,
      CALL_BUTTON_STUCK = 21,
      SELF_TEST_FAILURE = 22,
      RUNTIME_LIMIT_EXCEEDED = 23,
      POSITION_LOST = 24,
      LOAD_MEASUREMENT_FAULT = 25,
      OVERCAPACITY = 26,
      SHUTDOWN_FAULT = 27,
      HANDRAIL_FAULT = 28,
      STEPS_FAULT = 29,
    }
  }


  export class Journey extends jspb.Message {
    getStart(): Transport.Location | undefined;
    setStart(value?: Transport.Location): Journey;
    hasStart(): boolean;
    clearStart(): Journey;

    getDestinationsList(): Array<Transport.Location>;
    setDestinationsList(value: Array<Transport.Location>): Journey;
    clearDestinationsList(): Journey;
    addDestinations(value?: Transport.Location, index?: number): Transport.Location;

    getReason(): string;
    setReason(value: string): Journey;

    getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): Journey;
    hasStartTime(): boolean;
    clearStartTime(): Journey;

    getEstimatedArrivalTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setEstimatedArrivalTime(value?: google_protobuf_timestamp_pb.Timestamp): Journey;
    hasEstimatedArrivalTime(): boolean;
    clearEstimatedArrivalTime(): Journey;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Journey.AsObject;
    static toObject(includeInstance: boolean, msg: Journey): Journey.AsObject;
    static serializeBinaryToWriter(message: Journey, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Journey;
    static deserializeBinaryFromReader(message: Journey, reader: jspb.BinaryReader): Journey;
  }

  export namespace Journey {
    export type AsObject = {
      start?: Transport.Location.AsObject;
      destinationsList: Array<Transport.Location.AsObject>;
      reason: string;
      startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
      estimatedArrivalTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };
  }


  export class Door extends jspb.Message {
    getTitle(): string;
    setTitle(value: string): Door;

    getDeck(): number;
    setDeck(value: number): Door;

    getStatus(): Transport.Door.DoorStatus;
    setStatus(value: Transport.Door.DoorStatus): Door;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Door.AsObject;
    static toObject(includeInstance: boolean, msg: Door): Door.AsObject;
    static serializeBinaryToWriter(message: Door, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Door;
    static deserializeBinaryFromReader(message: Door, reader: jspb.BinaryReader): Door;
  }

  export namespace Door {
    export type AsObject = {
      title: string;
      deck: number;
      status: Transport.Door.DoorStatus;
    };

    export enum DoorStatus {
      DOOR_STATUS_UNSPECIFIED = 0,
      CLOSED = 1,
      OPEN = 2,
      OPENING = 3,
      CLOSING = 4,
      SAFETY_LOCKED = 5,
      LIMITED_OPENED = 6,
    }
  }


  export class Location extends jspb.Message {
    getId(): string;
    setId(value: string): Location;

    getTitle(): string;
    setTitle(value: string): Location;

    getDescription(): string;
    setDescription(value: string): Location;

    getFloor(): string;
    setFloor(value: string): Location;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Location.AsObject;
    static toObject(includeInstance: boolean, msg: Location): Location.AsObject;
    static serializeBinaryToWriter(message: Location, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Location;
    static deserializeBinaryFromReader(message: Location, reader: jspb.BinaryReader): Location;
  }

  export namespace Location {
    export type AsObject = {
      id: string;
      title: string;
      description: string;
      floor: string;
    };
  }


  export class StoppedReason extends jspb.Message {
    getReason(): Transport.StoppedReason.Reason;
    setReason(value: Transport.StoppedReason.Reason): StoppedReason;

    getTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setTime(value?: google_protobuf_timestamp_pb.Timestamp): StoppedReason;
    hasTime(): boolean;
    clearTime(): StoppedReason;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): StoppedReason.AsObject;
    static toObject(includeInstance: boolean, msg: StoppedReason): StoppedReason.AsObject;
    static serializeBinaryToWriter(message: StoppedReason, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): StoppedReason;
    static deserializeBinaryFromReader(message: StoppedReason, reader: jspb.BinaryReader): StoppedReason;
  }

  export namespace StoppedReason {
    export type AsObject = {
      reason: Transport.StoppedReason.Reason;
      time?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };

    export enum Reason {
      REASON_UNSPECIFIED = 0,
      EMERGENCY_STOP_SENSOR = 1,
      EMERGENCY_STOP_USER = 2,
      REMOTE_STOP = 3,
    }
  }


  export class Payload extends jspb.Message {
    getPayloadId(): string;
    setPayloadId(value: string): Payload;

    getDescription(): string;
    setDescription(value: string): Payload;

    getIntendedJourney(): Transport.Journey | undefined;
    setIntendedJourney(value?: Transport.Journey): Payload;
    hasIntendedJourney(): boolean;
    clearIntendedJourney(): Payload;

    getActualJourney(): Transport.Journey | undefined;
    setActualJourney(value?: Transport.Journey): Payload;
    hasActualJourney(): boolean;
    clearActualJourney(): Payload;

    getExternalIdsMap(): jspb.Map<string, string>;
    clearExternalIdsMap(): Payload;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Payload.AsObject;
    static toObject(includeInstance: boolean, msg: Payload): Payload.AsObject;
    static serializeBinaryToWriter(message: Payload, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Payload;
    static deserializeBinaryFromReader(message: Payload, reader: jspb.BinaryReader): Payload;
  }

  export namespace Payload {
    export type AsObject = {
      payloadId: string;
      description: string;
      intendedJourney?: Transport.Journey.AsObject;
      actualJourney?: Transport.Journey.AsObject;
      externalIdsMap: Array<[string, string]>;
    };
  }


  export enum Direction {
    DIRECTION_UNSPECIFIED = 0,
    NO_DIRECTION = 1,
    UP = 2,
    DOWN = 3,
    IN = 4,
    OUT = 5,
    CLOCKWISE = 6,
    ANTICLOCKWISE = 7,
    FORWARD = 8,
    BACKWARD = 9,
    EAST = 10,
    WEST = 11,
    NORTH = 12,
    SOUTH = 13,
    LEFT = 14,
    RIGHT = 15,
    SIDEWAYS = 16,
  }

  export enum OperatingMode {
    OPERATING_MODE_UNSPECIFIED = 0,
    NORMAL = 1,
    SERVICE_CONTROL = 2,
    FIREFIGHTER_CONTROL = 3,
    OUT_OF_SERVICE = 4,
    EMERGENCY_POWER = 5,
    VIP_CONTROL = 6,
    EARTHQUAKE_OPERATION = 7,
    FIRE_OPERATION = 8,
    ATTENDANT_CONTROL = 9,
    PARKING = 10,
    HOMING = 11,
    CABINET_RECALL = 12,
    OCCUPANT_EVACUATION = 13,
    FREIGHT = 14,
    FAILURE = 15,
    REDUCED_SPEED = 16,
    STORM_OPERATION = 17,
    HIGH_WIND_OPERATION = 18,
    AUTOMATIC = 19,
    CONTINUOUS = 20,
    ENERGY_SAVING = 21,
  }

  export enum Active {
    ACTIVE_UNSPECIFIED = 0,
    INACTIVE = 1,
    ACTIVE = 2,
    STANDBY = 3,
  }

  export enum LoadCase {
    _LOAD_NOT_SET = 0,
    LOAD = 4,
  }

  export enum SpeedCase {
    _SPEED_NOT_SET = 0,
    SPEED = 9,
  }
}

export class TransportSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): TransportSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): TransportSupport;

  getLoadUnit(): string;
  setLoadUnit(value: string): TransportSupport;

  getMaxLoad(): number;
  setMaxLoad(value: number): TransportSupport;

  getSpeedUnit(): string;
  setSpeedUnit(value: string): TransportSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TransportSupport.AsObject;
  static toObject(includeInstance: boolean, msg: TransportSupport): TransportSupport.AsObject;
  static serializeBinaryToWriter(message: TransportSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TransportSupport;
  static deserializeBinaryFromReader(message: TransportSupport, reader: jspb.BinaryReader): TransportSupport;
}

export namespace TransportSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    loadUnit: string;
    maxLoad: number;
    speedUnit: string;
  };
}

export class GetTransportRequest extends jspb.Message {
  getName(): string;
  setName(value: string): GetTransportRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): GetTransportRequest;
  hasReadMask(): boolean;
  clearReadMask(): GetTransportRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetTransportRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetTransportRequest): GetTransportRequest.AsObject;
  static serializeBinaryToWriter(message: GetTransportRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetTransportRequest;
  static deserializeBinaryFromReader(message: GetTransportRequest, reader: jspb.BinaryReader): GetTransportRequest;
}

export namespace GetTransportRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class DescribeTransportRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeTransportRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeTransportRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeTransportRequest): DescribeTransportRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeTransportRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeTransportRequest;
  static deserializeBinaryFromReader(message: DescribeTransportRequest, reader: jspb.BinaryReader): DescribeTransportRequest;
}

export namespace DescribeTransportRequest {
  export type AsObject = {
    name: string;
  };
}

export class PullTransportRequest extends jspb.Message {
  getName(): string;
  setName(value: string): PullTransportRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): PullTransportRequest;
  hasReadMask(): boolean;
  clearReadMask(): PullTransportRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): PullTransportRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullTransportRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PullTransportRequest): PullTransportRequest.AsObject;
  static serializeBinaryToWriter(message: PullTransportRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullTransportRequest;
  static deserializeBinaryFromReader(message: PullTransportRequest, reader: jspb.BinaryReader): PullTransportRequest;
}

export namespace PullTransportRequest {
  export type AsObject = {
    name: string;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class PullTransportResponse extends jspb.Message {
  getChangesList(): Array<PullTransportResponse.Change>;
  setChangesList(value: Array<PullTransportResponse.Change>): PullTransportResponse;
  clearChangesList(): PullTransportResponse;
  addChanges(value?: PullTransportResponse.Change, index?: number): PullTransportResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullTransportResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullTransportResponse): PullTransportResponse.AsObject;
  static serializeBinaryToWriter(message: PullTransportResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullTransportResponse;
  static deserializeBinaryFromReader(message: PullTransportResponse, reader: jspb.BinaryReader): PullTransportResponse;
}

export namespace PullTransportResponse {
  export type AsObject = {
    changesList: Array<PullTransportResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getChangeTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setChangeTime(value?: google_protobuf_timestamp_pb.Timestamp): Change;
    hasChangeTime(): boolean;
    clearChangeTime(): Change;

    getTransport(): Transport | undefined;
    setTransport(value?: Transport): Change;
    hasTransport(): boolean;
    clearTransport(): Change;

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
      transport?: Transport.AsObject;
    };
  }

}

