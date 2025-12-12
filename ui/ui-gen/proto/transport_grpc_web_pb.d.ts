import * as grpcWeb from 'grpc-web';

import * as transport_pb from './transport_pb'; // proto import: "transport.proto"


export class TransportApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getTransport(
    request: transport_pb.GetTransportRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: transport_pb.Transport) => void
  ): grpcWeb.ClientReadableStream<transport_pb.Transport>;

  pullTransport(
    request: transport_pb.PullTransportRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<transport_pb.PullTransportResponse>;

}

export class TransportInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeTransport(
    request: transport_pb.DescribeTransportRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: transport_pb.TransportSupport) => void
  ): grpcWeb.ClientReadableStream<transport_pb.TransportSupport>;

}

export class TransportApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getTransport(
    request: transport_pb.GetTransportRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<transport_pb.Transport>;

  pullTransport(
    request: transport_pb.PullTransportRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<transport_pb.PullTransportResponse>;

}

export class TransportInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeTransport(
    request: transport_pb.DescribeTransportRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<transport_pb.TransportSupport>;

}

