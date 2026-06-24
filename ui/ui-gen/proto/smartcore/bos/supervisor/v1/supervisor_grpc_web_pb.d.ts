import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_supervisor_v1_supervisor_pb from '../../../../smartcore/bos/supervisor/v1/supervisor_pb'; // proto import: "smartcore/bos/supervisor/v1/supervisor.proto"


export class SupervisorApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  installUpdate(
    request: smartcore_bos_supervisor_v1_supervisor_pb.InstallUpdateRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_supervisor_v1_supervisor_pb.InstallUpdateResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_supervisor_v1_supervisor_pb.InstallUpdateResponse>;

  getUpdateStatus(
    request: smartcore_bos_supervisor_v1_supervisor_pb.GetUpdateStatusRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_supervisor_v1_supervisor_pb.GetUpdateStatusResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_supervisor_v1_supervisor_pb.GetUpdateStatusResponse>;

  commit(
    request: smartcore_bos_supervisor_v1_supervisor_pb.CommitRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_supervisor_v1_supervisor_pb.CommitResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_supervisor_v1_supervisor_pb.CommitResponse>;

  installSupervisorUpdate(
    request: smartcore_bos_supervisor_v1_supervisor_pb.InstallSupervisorUpdateRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_supervisor_v1_supervisor_pb.InstallSupervisorUpdateResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_supervisor_v1_supervisor_pb.InstallSupervisorUpdateResponse>;

  getSupervisorInfo(
    request: smartcore_bos_supervisor_v1_supervisor_pb.GetSupervisorInfoRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_supervisor_v1_supervisor_pb.GetSupervisorInfoResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_supervisor_v1_supervisor_pb.GetSupervisorInfoResponse>;

}

export class SupervisorApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  installUpdate(
    request: smartcore_bos_supervisor_v1_supervisor_pb.InstallUpdateRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_supervisor_v1_supervisor_pb.InstallUpdateResponse>;

  getUpdateStatus(
    request: smartcore_bos_supervisor_v1_supervisor_pb.GetUpdateStatusRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_supervisor_v1_supervisor_pb.GetUpdateStatusResponse>;

  commit(
    request: smartcore_bos_supervisor_v1_supervisor_pb.CommitRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_supervisor_v1_supervisor_pb.CommitResponse>;

  installSupervisorUpdate(
    request: smartcore_bos_supervisor_v1_supervisor_pb.InstallSupervisorUpdateRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_supervisor_v1_supervisor_pb.InstallSupervisorUpdateResponse>;

  getSupervisorInfo(
    request: smartcore_bos_supervisor_v1_supervisor_pb.GetSupervisorInfoRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_supervisor_v1_supervisor_pb.GetSupervisorInfoResponse>;

}

