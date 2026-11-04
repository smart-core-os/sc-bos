import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_setup_v1_setup_pb from '../../../../smartcore/bos/setup/v1/setup_pb'; // proto import: "smartcore/bos/setup/v1/setup.proto"


export class SetupApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getSetupState(
    request: smartcore_bos_setup_v1_setup_pb.GetSetupStateRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_setup_v1_setup_pb.SetupState) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_setup_v1_setup_pb.SetupState>;

  completeSetup(
    request: smartcore_bos_setup_v1_setup_pb.CompleteSetupRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_setup_v1_setup_pb.CompleteSetupResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_setup_v1_setup_pb.CompleteSetupResponse>;

}

export class SetupApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getSetupState(
    request: smartcore_bos_setup_v1_setup_pb.GetSetupStateRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_setup_v1_setup_pb.SetupState>;

  completeSetup(
    request: smartcore_bos_setup_v1_setup_pb.CompleteSetupRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_setup_v1_setup_pb.CompleteSetupResponse>;

}

