import * as grpcWeb from 'grpc-web';

import * as traits_input_select_pb from '../traits/input_select_pb'; // proto import: "traits/input_select.proto"


export class InputSelectApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  updateInput(
    request: traits_input_select_pb.UpdateInputRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_input_select_pb.Input) => void
  ): grpcWeb.ClientReadableStream<traits_input_select_pb.Input>;

  getInput(
    request: traits_input_select_pb.GetInputRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_input_select_pb.Input) => void
  ): grpcWeb.ClientReadableStream<traits_input_select_pb.Input>;

  pullInput(
    request: traits_input_select_pb.PullInputRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_input_select_pb.PullInputResponse>;

}

export class InputSelectInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeInput(
    request: traits_input_select_pb.DescribeInputRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_input_select_pb.InputSupport) => void
  ): grpcWeb.ClientReadableStream<traits_input_select_pb.InputSupport>;

}

export class InputSelectApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  updateInput(
    request: traits_input_select_pb.UpdateInputRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_input_select_pb.Input>;

  getInput(
    request: traits_input_select_pb.GetInputRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_input_select_pb.Input>;

  pullInput(
    request: traits_input_select_pb.PullInputRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_input_select_pb.PullInputResponse>;

}

export class InputSelectInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeInput(
    request: traits_input_select_pb.DescribeInputRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_input_select_pb.InputSupport>;

}

