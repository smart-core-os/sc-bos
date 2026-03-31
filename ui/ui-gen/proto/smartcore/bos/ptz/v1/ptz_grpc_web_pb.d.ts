import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_ptz_v1_ptz_pb from '../../../../smartcore/bos/ptz/v1/ptz_pb'; // proto import: "smartcore/bos/ptz/v1/ptz.proto"


export class PtzApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPtz(
    request: smartcore_bos_ptz_v1_ptz_pb.GetPtzRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_ptz_v1_ptz_pb.Ptz) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_ptz_v1_ptz_pb.Ptz>;

  updatePtz(
    request: smartcore_bos_ptz_v1_ptz_pb.UpdatePtzRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_ptz_v1_ptz_pb.Ptz) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_ptz_v1_ptz_pb.Ptz>;

  stop(
    request: smartcore_bos_ptz_v1_ptz_pb.StopPtzRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_ptz_v1_ptz_pb.Ptz) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_ptz_v1_ptz_pb.Ptz>;

  createPreset(
    request: smartcore_bos_ptz_v1_ptz_pb.CreatePtzPresetRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_ptz_v1_ptz_pb.PtzPreset) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_ptz_v1_ptz_pb.PtzPreset>;

  pullPtz(
    request: smartcore_bos_ptz_v1_ptz_pb.PullPtzRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_ptz_v1_ptz_pb.PullPtzResponse>;

}

export class PtzInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describePtz(
    request: smartcore_bos_ptz_v1_ptz_pb.DescribePtzRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_ptz_v1_ptz_pb.PtzSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_ptz_v1_ptz_pb.PtzSupport>;

}

export class PtzApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPtz(
    request: smartcore_bos_ptz_v1_ptz_pb.GetPtzRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_ptz_v1_ptz_pb.Ptz>;

  updatePtz(
    request: smartcore_bos_ptz_v1_ptz_pb.UpdatePtzRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_ptz_v1_ptz_pb.Ptz>;

  stop(
    request: smartcore_bos_ptz_v1_ptz_pb.StopPtzRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_ptz_v1_ptz_pb.Ptz>;

  createPreset(
    request: smartcore_bos_ptz_v1_ptz_pb.CreatePtzPresetRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_ptz_v1_ptz_pb.PtzPreset>;

  pullPtz(
    request: smartcore_bos_ptz_v1_ptz_pb.PullPtzRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_ptz_v1_ptz_pb.PullPtzResponse>;

}

export class PtzInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describePtz(
    request: smartcore_bos_ptz_v1_ptz_pb.DescribePtzRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_ptz_v1_ptz_pb.PtzSupport>;

}

