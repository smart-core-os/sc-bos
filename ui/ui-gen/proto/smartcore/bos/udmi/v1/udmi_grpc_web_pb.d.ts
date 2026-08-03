import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_udmi_v1_udmi_pb from '../../../../smartcore/bos/udmi/v1/udmi_pb'; // proto import: "smartcore/bos/udmi/v1/udmi.proto"


export class UdmiServiceClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  pullControlTopics(
    request: smartcore_bos_udmi_v1_udmi_pb.PullControlTopicsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_udmi_v1_udmi_pb.PullControlTopicsResponse>;

  onMessage(
    request: smartcore_bos_udmi_v1_udmi_pb.OnMessageRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_udmi_v1_udmi_pb.OnMessageResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_udmi_v1_udmi_pb.OnMessageResponse>;

  pullExportMessages(
    request: smartcore_bos_udmi_v1_udmi_pb.PullExportMessagesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_udmi_v1_udmi_pb.PullExportMessagesResponse>;

  getExportMessage(
    request: smartcore_bos_udmi_v1_udmi_pb.GetExportMessageRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_udmi_v1_udmi_pb.MqttMessage) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_udmi_v1_udmi_pb.MqttMessage>;

}

export class UdmiExportApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listExportedPoints(
    request: smartcore_bos_udmi_v1_udmi_pb.ListExportedPointsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_udmi_v1_udmi_pb.ListExportedPointsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_udmi_v1_udmi_pb.ListExportedPointsResponse>;

}

export class UdmiServicePromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  pullControlTopics(
    request: smartcore_bos_udmi_v1_udmi_pb.PullControlTopicsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_udmi_v1_udmi_pb.PullControlTopicsResponse>;

  onMessage(
    request: smartcore_bos_udmi_v1_udmi_pb.OnMessageRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_udmi_v1_udmi_pb.OnMessageResponse>;

  pullExportMessages(
    request: smartcore_bos_udmi_v1_udmi_pb.PullExportMessagesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_udmi_v1_udmi_pb.PullExportMessagesResponse>;

  getExportMessage(
    request: smartcore_bos_udmi_v1_udmi_pb.GetExportMessageRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_udmi_v1_udmi_pb.MqttMessage>;

}

export class UdmiExportApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listExportedPoints(
    request: smartcore_bos_udmi_v1_udmi_pb.ListExportedPointsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_udmi_v1_udmi_pb.ListExportedPointsResponse>;

}

