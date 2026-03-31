import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_mqtt_v1_mqtt_pb from '../../../../smartcore/bos/mqtt/v1/mqtt_pb'; // proto import: "smartcore/bos/mqtt/v1/mqtt.proto"


export class MqttServiceClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  pullMessages(
    request: smartcore_bos_mqtt_v1_mqtt_pb.PullMessagesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_mqtt_v1_mqtt_pb.PullMessagesResponse>;

}

export class MqttServicePromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  pullMessages(
    request: smartcore_bos_mqtt_v1_mqtt_pb.PullMessagesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_mqtt_v1_mqtt_pb.PullMessagesResponse>;

}

