import * as jspb from 'google-protobuf'

import * as google_protobuf_field_mask_pb from 'google-protobuf/google/protobuf/field_mask_pb'; // proto import: "google/protobuf/field_mask.proto"
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb'; // proto import: "google/protobuf/timestamp.proto"
import * as types_change_pb from '../types/change_pb'; // proto import: "types/change.proto"
import * as types_info_pb from '../types/info_pb'; // proto import: "types/info.proto"
import * as types_time_period_pb from '../types/time/period_pb'; // proto import: "types/time/period.proto"
import * as types_time_unit_pb from '../types/time/unit_pb'; // proto import: "types/time/unit.proto"


export class Booking extends jspb.Message {
  getBookable(): string;
  setBookable(value: string): Booking;

  getId(): string;
  setId(value: string): Booking;

  getTitle(): string;
  setTitle(value: string): Booking;

  getOwnerName(): string;
  setOwnerName(value: string): Booking;

  getBooked(): types_time_period_pb.Period | undefined;
  setBooked(value?: types_time_period_pb.Period): Booking;
  hasBooked(): boolean;
  clearBooked(): Booking;

  getCheckIn(): types_time_period_pb.Period | undefined;
  setCheckIn(value?: types_time_period_pb.Period): Booking;
  hasCheckIn(): boolean;
  clearCheckIn(): Booking;

  getCheckInNotRequired(): boolean;
  setCheckInNotRequired(value: boolean): Booking;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Booking.AsObject;
  static toObject(includeInstance: boolean, msg: Booking): Booking.AsObject;
  static serializeBinaryToWriter(message: Booking, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Booking;
  static deserializeBinaryFromReader(message: Booking, reader: jspb.BinaryReader): Booking;
}

export namespace Booking {
  export type AsObject = {
    bookable: string;
    id: string;
    title: string;
    ownerName: string;
    booked?: types_time_period_pb.Period.AsObject;
    checkIn?: types_time_period_pb.Period.AsObject;
    checkInNotRequired: boolean;
  };
}

export class BookingSupport extends jspb.Message {
  getResourceSupport(): types_info_pb.ResourceSupport | undefined;
  setResourceSupport(value?: types_info_pb.ResourceSupport): BookingSupport;
  hasResourceSupport(): boolean;
  clearResourceSupport(): BookingSupport;

  getCheckInSupport(): BookingSupport.CheckInSupport;
  setCheckInSupport(value: BookingSupport.CheckInSupport): BookingSupport;

  getCheckOutSupport(): BookingSupport.CheckInSupport;
  setCheckOutSupport(value: BookingSupport.CheckInSupport): BookingSupport;

  getTimeResolution(): types_time_unit_pb.Unit;
  setTimeResolution(value: types_time_unit_pb.Unit): BookingSupport;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BookingSupport.AsObject;
  static toObject(includeInstance: boolean, msg: BookingSupport): BookingSupport.AsObject;
  static serializeBinaryToWriter(message: BookingSupport, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BookingSupport;
  static deserializeBinaryFromReader(message: BookingSupport, reader: jspb.BinaryReader): BookingSupport;
}

export namespace BookingSupport {
  export type AsObject = {
    resourceSupport?: types_info_pb.ResourceSupport.AsObject;
    checkInSupport: BookingSupport.CheckInSupport;
    checkOutSupport: BookingSupport.CheckInSupport;
    timeResolution: types_time_unit_pb.Unit;
  };

  export enum CheckInSupport {
    CHECK_IN_SUPPORT_UNSPECIFIED = 0,
    NO_SUPPORT = 1,
    STATE = 2,
    TIME = 3,
  }
}

export class ListBookingsRequest extends jspb.Message {
  getName(): string;
  setName(value: string): ListBookingsRequest;

  getBookingIntersects(): types_time_period_pb.Period | undefined;
  setBookingIntersects(value?: types_time_period_pb.Period): ListBookingsRequest;
  hasBookingIntersects(): boolean;
  clearBookingIntersects(): ListBookingsRequest;

  getReadMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setReadMask(value?: google_protobuf_field_mask_pb.FieldMask): ListBookingsRequest;
  hasReadMask(): boolean;
  clearReadMask(): ListBookingsRequest;

  getUpdatesOnly(): boolean;
  setUpdatesOnly(value: boolean): ListBookingsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListBookingsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: ListBookingsRequest): ListBookingsRequest.AsObject;
  static serializeBinaryToWriter(message: ListBookingsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListBookingsRequest;
  static deserializeBinaryFromReader(message: ListBookingsRequest, reader: jspb.BinaryReader): ListBookingsRequest;
}

export namespace ListBookingsRequest {
  export type AsObject = {
    name: string;
    bookingIntersects?: types_time_period_pb.Period.AsObject;
    readMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
    updatesOnly: boolean;
  };
}

export class ListBookingsResponse extends jspb.Message {
  getBookingsList(): Array<Booking>;
  setBookingsList(value: Array<Booking>): ListBookingsResponse;
  clearBookingsList(): ListBookingsResponse;
  addBookings(value?: Booking, index?: number): Booking;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ListBookingsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ListBookingsResponse): ListBookingsResponse.AsObject;
  static serializeBinaryToWriter(message: ListBookingsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ListBookingsResponse;
  static deserializeBinaryFromReader(message: ListBookingsResponse, reader: jspb.BinaryReader): ListBookingsResponse;
}

export namespace ListBookingsResponse {
  export type AsObject = {
    bookingsList: Array<Booking.AsObject>;
  };
}

export class CheckInBookingRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CheckInBookingRequest;

  getBookingId(): string;
  setBookingId(value: string): CheckInBookingRequest;

  getTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTime(value?: google_protobuf_timestamp_pb.Timestamp): CheckInBookingRequest;
  hasTime(): boolean;
  clearTime(): CheckInBookingRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CheckInBookingRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CheckInBookingRequest): CheckInBookingRequest.AsObject;
  static serializeBinaryToWriter(message: CheckInBookingRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CheckInBookingRequest;
  static deserializeBinaryFromReader(message: CheckInBookingRequest, reader: jspb.BinaryReader): CheckInBookingRequest;
}

export namespace CheckInBookingRequest {
  export type AsObject = {
    name: string;
    bookingId: string;
    time?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class CheckInBookingResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CheckInBookingResponse.AsObject;
  static toObject(includeInstance: boolean, msg: CheckInBookingResponse): CheckInBookingResponse.AsObject;
  static serializeBinaryToWriter(message: CheckInBookingResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CheckInBookingResponse;
  static deserializeBinaryFromReader(message: CheckInBookingResponse, reader: jspb.BinaryReader): CheckInBookingResponse;
}

export namespace CheckInBookingResponse {
  export type AsObject = {
  };
}

export class CheckOutBookingRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CheckOutBookingRequest;

  getBookingId(): string;
  setBookingId(value: string): CheckOutBookingRequest;

  getTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTime(value?: google_protobuf_timestamp_pb.Timestamp): CheckOutBookingRequest;
  hasTime(): boolean;
  clearTime(): CheckOutBookingRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CheckOutBookingRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CheckOutBookingRequest): CheckOutBookingRequest.AsObject;
  static serializeBinaryToWriter(message: CheckOutBookingRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CheckOutBookingRequest;
  static deserializeBinaryFromReader(message: CheckOutBookingRequest, reader: jspb.BinaryReader): CheckOutBookingRequest;
}

export namespace CheckOutBookingRequest {
  export type AsObject = {
    name: string;
    bookingId: string;
    time?: google_protobuf_timestamp_pb.Timestamp.AsObject;
  };
}

export class CheckOutBookingResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CheckOutBookingResponse.AsObject;
  static toObject(includeInstance: boolean, msg: CheckOutBookingResponse): CheckOutBookingResponse.AsObject;
  static serializeBinaryToWriter(message: CheckOutBookingResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CheckOutBookingResponse;
  static deserializeBinaryFromReader(message: CheckOutBookingResponse, reader: jspb.BinaryReader): CheckOutBookingResponse;
}

export namespace CheckOutBookingResponse {
  export type AsObject = {
  };
}

export class CreateBookingRequest extends jspb.Message {
  getName(): string;
  setName(value: string): CreateBookingRequest;

  getBooking(): Booking | undefined;
  setBooking(value?: Booking): CreateBookingRequest;
  hasBooking(): boolean;
  clearBooking(): CreateBookingRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateBookingRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CreateBookingRequest): CreateBookingRequest.AsObject;
  static serializeBinaryToWriter(message: CreateBookingRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateBookingRequest;
  static deserializeBinaryFromReader(message: CreateBookingRequest, reader: jspb.BinaryReader): CreateBookingRequest;
}

export namespace CreateBookingRequest {
  export type AsObject = {
    name: string;
    booking?: Booking.AsObject;
  };
}

export class CreateBookingResponse extends jspb.Message {
  getBookingId(): string;
  setBookingId(value: string): CreateBookingResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CreateBookingResponse.AsObject;
  static toObject(includeInstance: boolean, msg: CreateBookingResponse): CreateBookingResponse.AsObject;
  static serializeBinaryToWriter(message: CreateBookingResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CreateBookingResponse;
  static deserializeBinaryFromReader(message: CreateBookingResponse, reader: jspb.BinaryReader): CreateBookingResponse;
}

export namespace CreateBookingResponse {
  export type AsObject = {
    bookingId: string;
  };
}

export class UpdateBookingRequest extends jspb.Message {
  getName(): string;
  setName(value: string): UpdateBookingRequest;

  getBooking(): Booking | undefined;
  setBooking(value?: Booking): UpdateBookingRequest;
  hasBooking(): boolean;
  clearBooking(): UpdateBookingRequest;

  getUpdateMask(): google_protobuf_field_mask_pb.FieldMask | undefined;
  setUpdateMask(value?: google_protobuf_field_mask_pb.FieldMask): UpdateBookingRequest;
  hasUpdateMask(): boolean;
  clearUpdateMask(): UpdateBookingRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateBookingRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateBookingRequest): UpdateBookingRequest.AsObject;
  static serializeBinaryToWriter(message: UpdateBookingRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateBookingRequest;
  static deserializeBinaryFromReader(message: UpdateBookingRequest, reader: jspb.BinaryReader): UpdateBookingRequest;
}

export namespace UpdateBookingRequest {
  export type AsObject = {
    name: string;
    booking?: Booking.AsObject;
    updateMask?: google_protobuf_field_mask_pb.FieldMask.AsObject;
  };
}

export class UpdateBookingResponse extends jspb.Message {
  getBooking(): Booking | undefined;
  setBooking(value?: Booking): UpdateBookingResponse;
  hasBooking(): boolean;
  clearBooking(): UpdateBookingResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UpdateBookingResponse.AsObject;
  static toObject(includeInstance: boolean, msg: UpdateBookingResponse): UpdateBookingResponse.AsObject;
  static serializeBinaryToWriter(message: UpdateBookingResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UpdateBookingResponse;
  static deserializeBinaryFromReader(message: UpdateBookingResponse, reader: jspb.BinaryReader): UpdateBookingResponse;
}

export namespace UpdateBookingResponse {
  export type AsObject = {
    booking?: Booking.AsObject;
  };
}

export class PullBookingsResponse extends jspb.Message {
  getChangesList(): Array<PullBookingsResponse.Change>;
  setChangesList(value: Array<PullBookingsResponse.Change>): PullBookingsResponse;
  clearChangesList(): PullBookingsResponse;
  addChanges(value?: PullBookingsResponse.Change, index?: number): PullBookingsResponse.Change;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PullBookingsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PullBookingsResponse): PullBookingsResponse.AsObject;
  static serializeBinaryToWriter(message: PullBookingsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PullBookingsResponse;
  static deserializeBinaryFromReader(message: PullBookingsResponse, reader: jspb.BinaryReader): PullBookingsResponse;
}

export namespace PullBookingsResponse {
  export type AsObject = {
    changesList: Array<PullBookingsResponse.Change.AsObject>;
  };

  export class Change extends jspb.Message {
    getName(): string;
    setName(value: string): Change;

    getType(): types_change_pb.ChangeType;
    setType(value: types_change_pb.ChangeType): Change;

    getNewValue(): Booking | undefined;
    setNewValue(value?: Booking): Change;
    hasNewValue(): boolean;
    clearNewValue(): Change;

    getOldValue(): Booking | undefined;
    setOldValue(value?: Booking): Change;
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
      newValue?: Booking.AsObject;
      oldValue?: Booking.AsObject;
      changeTime?: google_protobuf_timestamp_pb.Timestamp.AsObject;
    };
  }

}

export class DescribeBookingRequest extends jspb.Message {
  getName(): string;
  setName(value: string): DescribeBookingRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DescribeBookingRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DescribeBookingRequest): DescribeBookingRequest.AsObject;
  static serializeBinaryToWriter(message: DescribeBookingRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DescribeBookingRequest;
  static deserializeBinaryFromReader(message: DescribeBookingRequest, reader: jspb.BinaryReader): DescribeBookingRequest;
}

export namespace DescribeBookingRequest {
  export type AsObject = {
    name: string;
  };
}

