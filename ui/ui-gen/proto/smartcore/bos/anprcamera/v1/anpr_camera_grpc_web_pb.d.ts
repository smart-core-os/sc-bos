import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_anprcamera_v1_anpr_camera_pb from '../../../../smartcore/bos/anprcamera/v1/anpr_camera_pb'; // proto import: "smartcore/bos/anprcamera/v1/anpr_camera.proto"


export class AnprCameraApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAnprEvents(
    request: smartcore_bos_anprcamera_v1_anpr_camera_pb.ListAnprEventsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_anprcamera_v1_anpr_camera_pb.ListAnprEventsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_anprcamera_v1_anpr_camera_pb.ListAnprEventsResponse>;

  pullAnprEvents(
    request: smartcore_bos_anprcamera_v1_anpr_camera_pb.PullAnprEventsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_anprcamera_v1_anpr_camera_pb.PullAnprEventsResponse>;

}

export class AnprCameraApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listAnprEvents(
    request: smartcore_bos_anprcamera_v1_anpr_camera_pb.ListAnprEventsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_anprcamera_v1_anpr_camera_pb.ListAnprEventsResponse>;

  pullAnprEvents(
    request: smartcore_bos_anprcamera_v1_anpr_camera_pb.PullAnprEventsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_anprcamera_v1_anpr_camera_pb.PullAnprEventsResponse>;

}

