import * as grpcWeb from 'grpc-web';

import * as traits_microphone_pb from '../traits/microphone_pb'; // proto import: "traits/microphone.proto"
import * as types_unit_pb from '../types/unit_pb'; // proto import: "types/unit.proto"


export class MicrophoneApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getGain(
    request: traits_microphone_pb.GetMicrophoneGainRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: types_unit_pb.AudioLevel) => void
  ): grpcWeb.ClientReadableStream<types_unit_pb.AudioLevel>;

  updateGain(
    request: traits_microphone_pb.UpdateMicrophoneGainRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: types_unit_pb.AudioLevel) => void
  ): grpcWeb.ClientReadableStream<types_unit_pb.AudioLevel>;

  pullGain(
    request: traits_microphone_pb.PullMicrophoneGainRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_microphone_pb.PullMicrophoneGainResponse>;

}

export class MicrophoneInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeGain(
    request: traits_microphone_pb.DescribeGainRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_microphone_pb.GainSupport) => void
  ): grpcWeb.ClientReadableStream<traits_microphone_pb.GainSupport>;

}

export class MicrophoneApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getGain(
    request: traits_microphone_pb.GetMicrophoneGainRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<types_unit_pb.AudioLevel>;

  updateGain(
    request: traits_microphone_pb.UpdateMicrophoneGainRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<types_unit_pb.AudioLevel>;

  pullGain(
    request: traits_microphone_pb.PullMicrophoneGainRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_microphone_pb.PullMicrophoneGainResponse>;

}

export class MicrophoneInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeGain(
    request: traits_microphone_pb.DescribeGainRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_microphone_pb.GainSupport>;

}

