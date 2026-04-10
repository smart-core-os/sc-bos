import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_booking_v1_booking_pb from '../../../../smartcore/bos/booking/v1/booking_pb'; // proto import: "smartcore/bos/booking/v1/booking.proto"


export class BookingApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listBookings(
    request: smartcore_bos_booking_v1_booking_pb.ListBookingsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_booking_v1_booking_pb.ListBookingsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_booking_v1_booking_pb.ListBookingsResponse>;

  checkInBooking(
    request: smartcore_bos_booking_v1_booking_pb.CheckInBookingRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_booking_v1_booking_pb.CheckInBookingResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_booking_v1_booking_pb.CheckInBookingResponse>;

  checkOutBooking(
    request: smartcore_bos_booking_v1_booking_pb.CheckOutBookingRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_booking_v1_booking_pb.CheckOutBookingResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_booking_v1_booking_pb.CheckOutBookingResponse>;

  createBooking(
    request: smartcore_bos_booking_v1_booking_pb.CreateBookingRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_booking_v1_booking_pb.CreateBookingResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_booking_v1_booking_pb.CreateBookingResponse>;

  updateBooking(
    request: smartcore_bos_booking_v1_booking_pb.UpdateBookingRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_booking_v1_booking_pb.UpdateBookingResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_booking_v1_booking_pb.UpdateBookingResponse>;

  pullBookings(
    request: smartcore_bos_booking_v1_booking_pb.ListBookingsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_booking_v1_booking_pb.PullBookingsResponse>;

}

export class BookingInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeBooking(
    request: smartcore_bos_booking_v1_booking_pb.DescribeBookingRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_booking_v1_booking_pb.BookingSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_booking_v1_booking_pb.BookingSupport>;

}

export class BookingApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listBookings(
    request: smartcore_bos_booking_v1_booking_pb.ListBookingsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_booking_v1_booking_pb.ListBookingsResponse>;

  checkInBooking(
    request: smartcore_bos_booking_v1_booking_pb.CheckInBookingRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_booking_v1_booking_pb.CheckInBookingResponse>;

  checkOutBooking(
    request: smartcore_bos_booking_v1_booking_pb.CheckOutBookingRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_booking_v1_booking_pb.CheckOutBookingResponse>;

  createBooking(
    request: smartcore_bos_booking_v1_booking_pb.CreateBookingRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_booking_v1_booking_pb.CreateBookingResponse>;

  updateBooking(
    request: smartcore_bos_booking_v1_booking_pb.UpdateBookingRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_booking_v1_booking_pb.UpdateBookingResponse>;

  pullBookings(
    request: smartcore_bos_booking_v1_booking_pb.ListBookingsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_booking_v1_booking_pb.PullBookingsResponse>;

}

export class BookingInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeBooking(
    request: smartcore_bos_booking_v1_booking_pb.DescribeBookingRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_booking_v1_booking_pb.BookingSupport>;

}

