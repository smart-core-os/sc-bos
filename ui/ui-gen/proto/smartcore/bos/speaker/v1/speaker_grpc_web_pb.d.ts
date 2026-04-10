import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_speaker_v1_speaker_pb from '../../../../smartcore/bos/speaker/v1/speaker_pb'; // proto import: "smartcore/bos/speaker/v1/speaker.proto"
import * as smartcore_bos_types_v1_unit_pb from '../../../../smartcore/bos/types/v1/unit_pb'; // proto import: "smartcore/bos/types/v1/unit.proto"


export class SpeakerApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getVolume(
    request: smartcore_bos_speaker_v1_speaker_pb.GetSpeakerVolumeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_types_v1_unit_pb.AudioLevel) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_types_v1_unit_pb.AudioLevel>;

  updateVolume(
    request: smartcore_bos_speaker_v1_speaker_pb.UpdateSpeakerVolumeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_types_v1_unit_pb.AudioLevel) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_types_v1_unit_pb.AudioLevel>;

  pullVolume(
    request: smartcore_bos_speaker_v1_speaker_pb.PullSpeakerVolumeRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_speaker_v1_speaker_pb.PullSpeakerVolumeResponse>;

}

export class SpeakerInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeVolume(
    request: smartcore_bos_speaker_v1_speaker_pb.DescribeVolumeRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_speaker_v1_speaker_pb.VolumeSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_speaker_v1_speaker_pb.VolumeSupport>;

}

export class SpeakerApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getVolume(
    request: smartcore_bos_speaker_v1_speaker_pb.GetSpeakerVolumeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_types_v1_unit_pb.AudioLevel>;

  updateVolume(
    request: smartcore_bos_speaker_v1_speaker_pb.UpdateSpeakerVolumeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_types_v1_unit_pb.AudioLevel>;

  pullVolume(
    request: smartcore_bos_speaker_v1_speaker_pb.PullSpeakerVolumeRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_speaker_v1_speaker_pb.PullSpeakerVolumeResponse>;

}

export class SpeakerInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeVolume(
    request: smartcore_bos_speaker_v1_speaker_pb.DescribeVolumeRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_speaker_v1_speaker_pb.VolumeSupport>;

}

