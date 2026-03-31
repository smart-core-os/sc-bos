import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_soundsensor_v1_sound_sensor_pb from '../../../../smartcore/bos/soundsensor/v1/sound_sensor_pb'; // proto import: "smartcore/bos/soundsensor/v1/sound_sensor.proto"


export class SoundSensorApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getSoundLevel(
    request: smartcore_bos_soundsensor_v1_sound_sensor_pb.GetSoundLevelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_soundsensor_v1_sound_sensor_pb.SoundLevel) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_soundsensor_v1_sound_sensor_pb.SoundLevel>;

  pullSoundLevel(
    request: smartcore_bos_soundsensor_v1_sound_sensor_pb.PullSoundLevelRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_soundsensor_v1_sound_sensor_pb.PullSoundLevelResponse>;

}

export class SoundSensorInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeSoundLevel(
    request: smartcore_bos_soundsensor_v1_sound_sensor_pb.DescribeSoundLevelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_soundsensor_v1_sound_sensor_pb.SoundLevelSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_soundsensor_v1_sound_sensor_pb.SoundLevelSupport>;

}

export class SoundSensorApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getSoundLevel(
    request: smartcore_bos_soundsensor_v1_sound_sensor_pb.GetSoundLevelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_soundsensor_v1_sound_sensor_pb.SoundLevel>;

  pullSoundLevel(
    request: smartcore_bos_soundsensor_v1_sound_sensor_pb.PullSoundLevelRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_soundsensor_v1_sound_sensor_pb.PullSoundLevelResponse>;

}

export class SoundSensorInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeSoundLevel(
    request: smartcore_bos_soundsensor_v1_sound_sensor_pb.DescribeSoundLevelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_soundsensor_v1_sound_sensor_pb.SoundLevelSupport>;

}

