import * as grpcWeb from 'grpc-web';

import * as traits_booking_pb from '../traits/booking_pb'; // proto import: "traits/booking.proto"


export class BookingApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listBookings(
    request: traits_booking_pb.ListBookingsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_booking_pb.ListBookingsResponse) => void
  ): grpcWeb.ClientReadableStream<traits_booking_pb.ListBookingsResponse>;

  checkInBooking(
    request: traits_booking_pb.CheckInBookingRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_booking_pb.CheckInBookingResponse) => void
  ): grpcWeb.ClientReadableStream<traits_booking_pb.CheckInBookingResponse>;

  checkOutBooking(
    request: traits_booking_pb.CheckOutBookingRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_booking_pb.CheckOutBookingResponse) => void
  ): grpcWeb.ClientReadableStream<traits_booking_pb.CheckOutBookingResponse>;

  createBooking(
    request: traits_booking_pb.CreateBookingRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_booking_pb.CreateBookingResponse) => void
  ): grpcWeb.ClientReadableStream<traits_booking_pb.CreateBookingResponse>;

  updateBooking(
    request: traits_booking_pb.UpdateBookingRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_booking_pb.UpdateBookingResponse) => void
  ): grpcWeb.ClientReadableStream<traits_booking_pb.UpdateBookingResponse>;

  pullBookings(
    request: traits_booking_pb.ListBookingsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_booking_pb.PullBookingsResponse>;

}

export class BookingInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeBooking(
    request: traits_booking_pb.DescribeBookingRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_booking_pb.BookingSupport) => void
  ): grpcWeb.ClientReadableStream<traits_booking_pb.BookingSupport>;

}

export class BookingApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listBookings(
    request: traits_booking_pb.ListBookingsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_booking_pb.ListBookingsResponse>;

  checkInBooking(
    request: traits_booking_pb.CheckInBookingRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_booking_pb.CheckInBookingResponse>;

  checkOutBooking(
    request: traits_booking_pb.CheckOutBookingRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_booking_pb.CheckOutBookingResponse>;

  createBooking(
    request: traits_booking_pb.CreateBookingRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_booking_pb.CreateBookingResponse>;

  updateBooking(
    request: traits_booking_pb.UpdateBookingRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_booking_pb.UpdateBookingResponse>;

  pullBookings(
    request: traits_booking_pb.ListBookingsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_booking_pb.PullBookingsResponse>;

}

export class BookingInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeBooking(
    request: traits_booking_pb.DescribeBookingRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_booking_pb.BookingSupport>;

}

