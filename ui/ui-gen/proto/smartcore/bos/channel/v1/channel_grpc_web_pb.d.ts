import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_channel_v1_channel_pb from '../../../../smartcore/bos/channel/v1/channel_pb'; // proto import: "smartcore/bos/channel/v1/channel.proto"


export class ChannelApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getChosenChannel(
    request: smartcore_bos_channel_v1_channel_pb.GetChosenChannelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_channel_v1_channel_pb.Channel) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_channel_v1_channel_pb.Channel>;

  chooseChannel(
    request: smartcore_bos_channel_v1_channel_pb.ChooseChannelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_channel_v1_channel_pb.Channel) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_channel_v1_channel_pb.Channel>;

  adjustChannel(
    request: smartcore_bos_channel_v1_channel_pb.AdjustChannelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_channel_v1_channel_pb.Channel) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_channel_v1_channel_pb.Channel>;

  returnChannel(
    request: smartcore_bos_channel_v1_channel_pb.ReturnChannelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_channel_v1_channel_pb.Channel) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_channel_v1_channel_pb.Channel>;

  pullChosenChannel(
    request: smartcore_bos_channel_v1_channel_pb.PullChosenChannelRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_channel_v1_channel_pb.PullChosenChannelResponse>;

}

export class ChannelInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeChosenChannel(
    request: smartcore_bos_channel_v1_channel_pb.DescribeChosenChannelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_channel_v1_channel_pb.ChosenChannelSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_channel_v1_channel_pb.ChosenChannelSupport>;

}

export class ChannelApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getChosenChannel(
    request: smartcore_bos_channel_v1_channel_pb.GetChosenChannelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_channel_v1_channel_pb.Channel>;

  chooseChannel(
    request: smartcore_bos_channel_v1_channel_pb.ChooseChannelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_channel_v1_channel_pb.Channel>;

  adjustChannel(
    request: smartcore_bos_channel_v1_channel_pb.AdjustChannelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_channel_v1_channel_pb.Channel>;

  returnChannel(
    request: smartcore_bos_channel_v1_channel_pb.ReturnChannelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_channel_v1_channel_pb.Channel>;

  pullChosenChannel(
    request: smartcore_bos_channel_v1_channel_pb.PullChosenChannelRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_channel_v1_channel_pb.PullChosenChannelResponse>;

}

export class ChannelInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeChosenChannel(
    request: smartcore_bos_channel_v1_channel_pb.DescribeChosenChannelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_channel_v1_channel_pb.ChosenChannelSupport>;

}

