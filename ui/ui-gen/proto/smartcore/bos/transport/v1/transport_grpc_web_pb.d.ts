import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_transport_v1_transport_pb from '../../../../smartcore/bos/transport/v1/transport_pb'; // proto import: "smartcore/bos/transport/v1/transport.proto"


export class TransportApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getTransport(
    request: smartcore_bos_transport_v1_transport_pb.GetTransportRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_transport_v1_transport_pb.Transport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_transport_v1_transport_pb.Transport>;

  pullTransport(
    request: smartcore_bos_transport_v1_transport_pb.PullTransportRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_transport_v1_transport_pb.PullTransportResponse>;

}

export class TransportInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeTransport(
    request: smartcore_bos_transport_v1_transport_pb.DescribeTransportRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_transport_v1_transport_pb.TransportSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_transport_v1_transport_pb.TransportSupport>;

}

export class TransportApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getTransport(
    request: smartcore_bos_transport_v1_transport_pb.GetTransportRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_transport_v1_transport_pb.Transport>;

  pullTransport(
    request: smartcore_bos_transport_v1_transport_pb.PullTransportRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_transport_v1_transport_pb.PullTransportResponse>;

}

export class TransportInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeTransport(
    request: smartcore_bos_transport_v1_transport_pb.DescribeTransportRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_transport_v1_transport_pb.TransportSupport>;

}

