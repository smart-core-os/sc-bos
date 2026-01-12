import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_serviceticket_v1_service_ticket_pb from '../../../../smartcore/bos/serviceticket/v1/service_ticket_pb'; // proto import: "smartcore/bos/serviceticket/v1/service_ticket.proto"


export class ServiceTicketApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createTicket(
    request: smartcore_bos_serviceticket_v1_service_ticket_pb.CreateTicketRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_serviceticket_v1_service_ticket_pb.Ticket) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_serviceticket_v1_service_ticket_pb.Ticket>;

  updateTicket(
    request: smartcore_bos_serviceticket_v1_service_ticket_pb.UpdateTicketRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_serviceticket_v1_service_ticket_pb.Ticket) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_serviceticket_v1_service_ticket_pb.Ticket>;

}

export class ServiceTicketInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeTicket(
    request: smartcore_bos_serviceticket_v1_service_ticket_pb.DescribeTicketRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_serviceticket_v1_service_ticket_pb.TicketSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_serviceticket_v1_service_ticket_pb.TicketSupport>;

}

export class ServiceTicketApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createTicket(
    request: smartcore_bos_serviceticket_v1_service_ticket_pb.CreateTicketRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_serviceticket_v1_service_ticket_pb.Ticket>;

  updateTicket(
    request: smartcore_bos_serviceticket_v1_service_ticket_pb.UpdateTicketRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_serviceticket_v1_service_ticket_pb.Ticket>;

}

export class ServiceTicketInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeTicket(
    request: smartcore_bos_serviceticket_v1_service_ticket_pb.DescribeTicketRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_serviceticket_v1_service_ticket_pb.TicketSupport>;

}

