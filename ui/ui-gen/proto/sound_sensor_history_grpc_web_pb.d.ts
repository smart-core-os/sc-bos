import * as grpcWeb from 'grpc-web';

import * as sound_sensor_history_pb from './sound_sensor_history_pb'; // proto import: "sound_sensor_history.proto"


export class SoundSensorHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listSoundLevelHistory(
    request: sound_sensor_history_pb.ListSoundLevelHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: sound_sensor_history_pb.ListSoundLevelHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<sound_sensor_history_pb.ListSoundLevelHistoryResponse>;

}

export class SoundSensorHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listSoundLevelHistory(
    request: sound_sensor_history_pb.ListSoundLevelHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<sound_sensor_history_pb.ListSoundLevelHistoryResponse>;

}

