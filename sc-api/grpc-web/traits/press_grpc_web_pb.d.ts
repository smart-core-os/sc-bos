import * as grpcWeb from 'grpc-web';

import * as traits_press_pb from '../traits/press_pb'; // proto import: "traits/press.proto"


export class PressApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPressedState(
    request: traits_press_pb.GetPressedStateRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_press_pb.PressedState) => void
  ): grpcWeb.ClientReadableStream<traits_press_pb.PressedState>;

  pullPressedState(
    request: traits_press_pb.PullPressedStateRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_press_pb.PullPressedStateResponse>;

  updatePressedState(
    request: traits_press_pb.UpdatePressedStateRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_press_pb.PressedState) => void
  ): grpcWeb.ClientReadableStream<traits_press_pb.PressedState>;

}

export class PressApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPressedState(
    request: traits_press_pb.GetPressedStateRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_press_pb.PressedState>;

  pullPressedState(
    request: traits_press_pb.PullPressedStateRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_press_pb.PullPressedStateResponse>;

  updatePressedState(
    request: traits_press_pb.UpdatePressedStateRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_press_pb.PressedState>;

}

