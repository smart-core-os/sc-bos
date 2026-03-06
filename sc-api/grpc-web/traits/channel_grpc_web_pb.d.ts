import * as grpcWeb from 'grpc-web';

import * as traits_channel_pb from '../traits/channel_pb'; // proto import: "traits/channel.proto"


export class ChannelApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getChosenChannel(
    request: traits_channel_pb.GetChosenChannelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_channel_pb.Channel) => void
  ): grpcWeb.ClientReadableStream<traits_channel_pb.Channel>;

  chooseChannel(
    request: traits_channel_pb.ChooseChannelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_channel_pb.Channel) => void
  ): grpcWeb.ClientReadableStream<traits_channel_pb.Channel>;

  adjustChannel(
    request: traits_channel_pb.AdjustChannelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_channel_pb.Channel) => void
  ): grpcWeb.ClientReadableStream<traits_channel_pb.Channel>;

  returnChannel(
    request: traits_channel_pb.ReturnChannelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_channel_pb.Channel) => void
  ): grpcWeb.ClientReadableStream<traits_channel_pb.Channel>;

  pullChosenChannel(
    request: traits_channel_pb.PullChosenChannelRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_channel_pb.PullChosenChannelResponse>;

}

export class ChannelInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeChosenChannel(
    request: traits_channel_pb.DescribeChosenChannelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_channel_pb.ChosenChannelSupport) => void
  ): grpcWeb.ClientReadableStream<traits_channel_pb.ChosenChannelSupport>;

}

export class ChannelApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getChosenChannel(
    request: traits_channel_pb.GetChosenChannelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_channel_pb.Channel>;

  chooseChannel(
    request: traits_channel_pb.ChooseChannelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_channel_pb.Channel>;

  adjustChannel(
    request: traits_channel_pb.AdjustChannelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_channel_pb.Channel>;

  returnChannel(
    request: traits_channel_pb.ReturnChannelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_channel_pb.Channel>;

  pullChosenChannel(
    request: traits_channel_pb.PullChosenChannelRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_channel_pb.PullChosenChannelResponse>;

}

export class ChannelInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeChosenChannel(
    request: traits_channel_pb.DescribeChosenChannelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_channel_pb.ChosenChannelSupport>;

}

