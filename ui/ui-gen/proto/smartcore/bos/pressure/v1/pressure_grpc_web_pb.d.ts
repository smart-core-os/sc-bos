import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_pressure_v1_pressure_pb from '../../../../smartcore/bos/pressure/v1/pressure_pb'; // proto import: "smartcore/bos/pressure/v1/pressure.proto"


export class PressureApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPressure(
    request: smartcore_bos_pressure_v1_pressure_pb.GetPressureRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_pressure_v1_pressure_pb.Pressure) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_pressure_v1_pressure_pb.Pressure>;

  pullPressure(
    request: smartcore_bos_pressure_v1_pressure_pb.PullPressureRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_pressure_v1_pressure_pb.PullPressureResponse>;

  updatePressure(
    request: smartcore_bos_pressure_v1_pressure_pb.UpdatePressureRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_pressure_v1_pressure_pb.Pressure) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_pressure_v1_pressure_pb.Pressure>;

}

export class PressureInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describePressure(
    request: smartcore_bos_pressure_v1_pressure_pb.DescribePressureRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_pressure_v1_pressure_pb.PressureSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_pressure_v1_pressure_pb.PressureSupport>;

}

export class PressureApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getPressure(
    request: smartcore_bos_pressure_v1_pressure_pb.GetPressureRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_pressure_v1_pressure_pb.Pressure>;

  pullPressure(
    request: smartcore_bos_pressure_v1_pressure_pb.PullPressureRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_pressure_v1_pressure_pb.PullPressureResponse>;

  updatePressure(
    request: smartcore_bos_pressure_v1_pressure_pb.UpdatePressureRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_pressure_v1_pressure_pb.Pressure>;

}

export class PressureInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describePressure(
    request: smartcore_bos_pressure_v1_pressure_pb.DescribePressureRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_pressure_v1_pressure_pb.PressureSupport>;

}

