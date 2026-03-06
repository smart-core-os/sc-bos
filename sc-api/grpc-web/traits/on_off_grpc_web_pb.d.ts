import * as grpcWeb from 'grpc-web';

import * as traits_on_off_pb from '../traits/on_off_pb'; // proto import: "traits/on_off.proto"


export class OnOffApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getOnOff(
    request: traits_on_off_pb.GetOnOffRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_on_off_pb.OnOff) => void
  ): grpcWeb.ClientReadableStream<traits_on_off_pb.OnOff>;

  updateOnOff(
    request: traits_on_off_pb.UpdateOnOffRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_on_off_pb.OnOff) => void
  ): grpcWeb.ClientReadableStream<traits_on_off_pb.OnOff>;

  pullOnOff(
    request: traits_on_off_pb.PullOnOffRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_on_off_pb.PullOnOffResponse>;

}

export class OnOffInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeOnOff(
    request: traits_on_off_pb.DescribeOnOffRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_on_off_pb.OnOffSupport) => void
  ): grpcWeb.ClientReadableStream<traits_on_off_pb.OnOffSupport>;

}

export class OnOffApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getOnOff(
    request: traits_on_off_pb.GetOnOffRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_on_off_pb.OnOff>;

  updateOnOff(
    request: traits_on_off_pb.UpdateOnOffRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_on_off_pb.OnOff>;

  pullOnOff(
    request: traits_on_off_pb.PullOnOffRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_on_off_pb.PullOnOffResponse>;

}

export class OnOffInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeOnOff(
    request: traits_on_off_pb.DescribeOnOffRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_on_off_pb.OnOffSupport>;

}

