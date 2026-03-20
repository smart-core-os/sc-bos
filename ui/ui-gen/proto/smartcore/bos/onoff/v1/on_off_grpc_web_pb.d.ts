import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_onoff_v1_on_off_pb from '../../../../smartcore/bos/onoff/v1/on_off_pb'; // proto import: "smartcore/bos/onoff/v1/on_off.proto"


export class OnOffApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getOnOff(
    request: smartcore_bos_onoff_v1_on_off_pb.GetOnOffRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_onoff_v1_on_off_pb.OnOff) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_onoff_v1_on_off_pb.OnOff>;

  updateOnOff(
    request: smartcore_bos_onoff_v1_on_off_pb.UpdateOnOffRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_onoff_v1_on_off_pb.OnOff) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_onoff_v1_on_off_pb.OnOff>;

  pullOnOff(
    request: smartcore_bos_onoff_v1_on_off_pb.PullOnOffRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_onoff_v1_on_off_pb.PullOnOffResponse>;

}

export class OnOffInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeOnOff(
    request: smartcore_bos_onoff_v1_on_off_pb.DescribeOnOffRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_onoff_v1_on_off_pb.OnOffSupport) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_onoff_v1_on_off_pb.OnOffSupport>;

}

export class OnOffApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getOnOff(
    request: smartcore_bos_onoff_v1_on_off_pb.GetOnOffRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_onoff_v1_on_off_pb.OnOff>;

  updateOnOff(
    request: smartcore_bos_onoff_v1_on_off_pb.UpdateOnOffRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_onoff_v1_on_off_pb.OnOff>;

  pullOnOff(
    request: smartcore_bos_onoff_v1_on_off_pb.PullOnOffRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_onoff_v1_on_off_pb.PullOnOffResponse>;

}

export class OnOffInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  describeOnOff(
    request: smartcore_bos_onoff_v1_on_off_pb.DescribeOnOffRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_onoff_v1_on_off_pb.OnOffSupport>;

}

