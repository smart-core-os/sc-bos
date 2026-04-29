import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_inputselect_v1_input_select_pb from '../../../../smartcore/bos/inputselect/v1/input_select_pb'; // proto import: "smartcore/bos/inputselect/v1/input_select.proto"


export class InputSelectApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  updateInput(
    request: smartcore_bos_inputselect_v1_input_select_pb.UpdateInputRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_inputselect_v1_input_select_pb.Input) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_inputselect_v1_input_select_pb.Input>;

  getInput(
    request: smartcore_bos_inputselect_v1_input_select_pb.GetInputRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_inputselect_v1_input_select_pb.Input) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_inputselect_v1_input_select_pb.Input>;

  pullInput(
    request: smartcore_bos_inputselect_v1_input_select_pb.PullInputRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_inputselect_v1_input_select_pb.PullInputResponse>;

}

export class InputSelectInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeInput(
    request: smartcore_bos_inputselect_v1_input_select_pb.DescribeInputRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_inputselect_v1_input_select_pb.InputSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_inputselect_v1_input_select_pb.InputSupport>;

}

export class InputSelectApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  updateInput(
    request: smartcore_bos_inputselect_v1_input_select_pb.UpdateInputRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_inputselect_v1_input_select_pb.Input>;

  getInput(
    request: smartcore_bos_inputselect_v1_input_select_pb.GetInputRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_inputselect_v1_input_select_pb.Input>;

  pullInput(
    request: smartcore_bos_inputselect_v1_input_select_pb.PullInputRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_inputselect_v1_input_select_pb.PullInputResponse>;

}

export class InputSelectInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeInput(
    request: smartcore_bos_inputselect_v1_input_select_pb.DescribeInputRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_inputselect_v1_input_select_pb.InputSupport>;

}

