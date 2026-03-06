import * as grpcWeb from 'grpc-web';

import * as traits_speaker_pb from '../traits/speaker_pb'; // proto import: "traits/speaker.proto"
import * as types_unit_pb from '../types/unit_pb'; // proto import: "types/unit.proto"


export class SpeakerApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getVolume(
    request: traits_speaker_pb.GetSpeakerVolumeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: types_unit_pb.AudioLevel) => void
  ): grpcWeb.ClientReadableStream<types_unit_pb.AudioLevel>;

  updateVolume(
    request: traits_speaker_pb.UpdateSpeakerVolumeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: types_unit_pb.AudioLevel) => void
  ): grpcWeb.ClientReadableStream<types_unit_pb.AudioLevel>;

  pullVolume(
    request: traits_speaker_pb.PullSpeakerVolumeRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_speaker_pb.PullSpeakerVolumeResponse>;

}

export class SpeakerInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeVolume(
    request: traits_speaker_pb.DescribeVolumeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_speaker_pb.VolumeSupport) => void
  ): grpcWeb.ClientReadableStream<traits_speaker_pb.VolumeSupport>;

}

export class SpeakerApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getVolume(
    request: traits_speaker_pb.GetSpeakerVolumeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<types_unit_pb.AudioLevel>;

  updateVolume(
    request: traits_speaker_pb.UpdateSpeakerVolumeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<types_unit_pb.AudioLevel>;

  pullVolume(
    request: traits_speaker_pb.PullSpeakerVolumeRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_speaker_pb.PullSpeakerVolumeResponse>;

}

export class SpeakerInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeVolume(
    request: traits_speaker_pb.DescribeVolumeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_speaker_pb.VolumeSupport>;

}

