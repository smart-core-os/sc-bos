import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_button_v1_button_pb from '../../../../smartcore/bos/button/v1/button_pb'; // proto import: "smartcore/bos/button/v1/button.proto"


export class ButtonApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getButtonState(
    request: smartcore_bos_button_v1_button_pb.GetButtonStateRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_button_v1_button_pb.ButtonState) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_button_v1_button_pb.ButtonState>;

  pullButtonState(
    request: smartcore_bos_button_v1_button_pb.PullButtonStateRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_button_v1_button_pb.PullButtonStateResponse>;

  updateButtonState(
    request: smartcore_bos_button_v1_button_pb.UpdateButtonStateRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_button_v1_button_pb.ButtonState) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_button_v1_button_pb.ButtonState>;

}

export class ButtonApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getButtonState(
    request: smartcore_bos_button_v1_button_pb.GetButtonStateRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_button_v1_button_pb.ButtonState>;

  pullButtonState(
    request: smartcore_bos_button_v1_button_pb.PullButtonStateRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_button_v1_button_pb.PullButtonStateResponse>;

  updateButtonState(
    request: smartcore_bos_button_v1_button_pb.UpdateButtonStateRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_button_v1_button_pb.ButtonState>;

}

