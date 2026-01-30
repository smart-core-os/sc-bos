import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_soundsensor_v1_sound_sensor_history_pb from '../../../../smartcore/bos/soundsensor/v1/sound_sensor_history_pb'; // proto import: "smartcore/bos/soundsensor/v1/sound_sensor_history.proto"


export class SoundSensorHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listSoundLevelHistory(
    request: smartcore_bos_soundsensor_v1_sound_sensor_history_pb.ListSoundLevelHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_soundsensor_v1_sound_sensor_history_pb.ListSoundLevelHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_soundsensor_v1_sound_sensor_history_pb.ListSoundLevelHistoryResponse>;

}

export class SoundSensorHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listSoundLevelHistory(
    request: smartcore_bos_soundsensor_v1_sound_sensor_history_pb.ListSoundLevelHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_soundsensor_v1_sound_sensor_history_pb.ListSoundLevelHistoryResponse>;

}

