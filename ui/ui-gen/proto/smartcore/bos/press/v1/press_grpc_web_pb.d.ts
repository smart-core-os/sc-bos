import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_press_v1_press_pb from '../../../../smartcore/bos/press/v1/press_pb'; // proto import: "smartcore/bos/press/v1/press.proto"


export class PressApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPressedState(
    request: smartcore_bos_press_v1_press_pb.GetPressedStateRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_press_v1_press_pb.PressedState) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_press_v1_press_pb.PressedState>;

  pullPressedState(
    request: smartcore_bos_press_v1_press_pb.PullPressedStateRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_press_v1_press_pb.PullPressedStateResponse>;

  updatePressedState(
    request: smartcore_bos_press_v1_press_pb.UpdatePressedStateRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_press_v1_press_pb.PressedState) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_press_v1_press_pb.PressedState>;

}

export class PressApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPressedState(
    request: smartcore_bos_press_v1_press_pb.GetPressedStateRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_press_v1_press_pb.PressedState>;

  pullPressedState(
    request: smartcore_bos_press_v1_press_pb.PullPressedStateRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_press_v1_press_pb.PullPressedStateResponse>;

  updatePressedState(
    request: smartcore_bos_press_v1_press_pb.UpdatePressedStateRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_press_v1_press_pb.PressedState>;

}

