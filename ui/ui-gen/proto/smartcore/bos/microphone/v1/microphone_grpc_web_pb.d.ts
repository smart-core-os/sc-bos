import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_microphone_v1_microphone_pb from '../../../../smartcore/bos/microphone/v1/microphone_pb'; // proto import: "smartcore/bos/microphone/v1/microphone.proto"
import * as smartcore_bos_types_v1_unit_pb from '../../../../smartcore/bos/types/v1/unit_pb'; // proto import: "smartcore/bos/types/v1/unit.proto"


export class MicrophoneApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getGain(
    request: smartcore_bos_microphone_v1_microphone_pb.GetMicrophoneGainRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_types_v1_unit_pb.AudioLevel) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_types_v1_unit_pb.AudioLevel>;

  updateGain(
    request: smartcore_bos_microphone_v1_microphone_pb.UpdateMicrophoneGainRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_types_v1_unit_pb.AudioLevel) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_types_v1_unit_pb.AudioLevel>;

  pullGain(
    request: smartcore_bos_microphone_v1_microphone_pb.PullMicrophoneGainRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_microphone_v1_microphone_pb.PullMicrophoneGainResponse>;

}

export class MicrophoneInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeGain(
    request: smartcore_bos_microphone_v1_microphone_pb.DescribeGainRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_microphone_v1_microphone_pb.GainSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_microphone_v1_microphone_pb.GainSupport>;

}

export class MicrophoneApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getGain(
    request: smartcore_bos_microphone_v1_microphone_pb.GetMicrophoneGainRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_types_v1_unit_pb.AudioLevel>;

  updateGain(
    request: smartcore_bos_microphone_v1_microphone_pb.UpdateMicrophoneGainRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_types_v1_unit_pb.AudioLevel>;

  pullGain(
    request: smartcore_bos_microphone_v1_microphone_pb.PullMicrophoneGainRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_microphone_v1_microphone_pb.PullMicrophoneGainResponse>;

}

export class MicrophoneInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeGain(
    request: smartcore_bos_microphone_v1_microphone_pb.DescribeGainRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_microphone_v1_microphone_pb.GainSupport>;

}

