import * as grpcWeb from 'grpc-web';

import * as traits_ptz_pb from '../traits/ptz_pb'; // proto import: "traits/ptz.proto"


export class PtzApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPtz(
    request: traits_ptz_pb.GetPtzRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_ptz_pb.Ptz) => void
  ): grpcWeb.ClientReadableStream<traits_ptz_pb.Ptz>;

  updatePtz(
    request: traits_ptz_pb.UpdatePtzRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_ptz_pb.Ptz) => void
  ): grpcWeb.ClientReadableStream<traits_ptz_pb.Ptz>;

  stop(
    request: traits_ptz_pb.StopPtzRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_ptz_pb.Ptz) => void
  ): grpcWeb.ClientReadableStream<traits_ptz_pb.Ptz>;

  createPreset(
    request: traits_ptz_pb.CreatePtzPresetRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_ptz_pb.PtzPreset) => void
  ): grpcWeb.ClientReadableStream<traits_ptz_pb.PtzPreset>;

  pullPtz(
    request: traits_ptz_pb.PullPtzRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_ptz_pb.PullPtzResponse>;

}

export class PtzInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describePtz(
    request: traits_ptz_pb.DescribePtzRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_ptz_pb.PtzSupport) => void
  ): grpcWeb.ClientReadableStream<traits_ptz_pb.PtzSupport>;

}

export class PtzApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPtz(
    request: traits_ptz_pb.GetPtzRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_ptz_pb.Ptz>;

  updatePtz(
    request: traits_ptz_pb.UpdatePtzRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_ptz_pb.Ptz>;

  stop(
    request: traits_ptz_pb.StopPtzRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_ptz_pb.Ptz>;

  createPreset(
    request: traits_ptz_pb.CreatePtzPresetRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_ptz_pb.PtzPreset>;

  pullPtz(
    request: traits_ptz_pb.PullPtzRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_ptz_pb.PullPtzResponse>;

}

export class PtzInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describePtz(
    request: traits_ptz_pb.DescribePtzRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_ptz_pb.PtzSupport>;

}

