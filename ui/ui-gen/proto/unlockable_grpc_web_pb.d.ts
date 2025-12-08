import * as grpcWeb from 'grpc-web';

import * as unlockable_pb from './unlockable_pb'; // proto import: "unlockable.proto"


export class UnlockableAPIClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listUnlockables(
    request: unlockable_pb.ListUnlockablesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: unlockable_pb.ListUnlockablesResponse) => void
  ): grpcWeb.ClientReadableStream<unlockable_pb.ListUnlockablesResponse>;

  unlockUnlockable(
    request: unlockable_pb.UnlockUnlockableRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: unlockable_pb.UnlockUnlockableResponse) => void
  ): grpcWeb.ClientReadableStream<unlockable_pb.UnlockUnlockableResponse>;

  lockUnlockable(
    request: unlockable_pb.LockUnlockableRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: unlockable_pb.LockUnlockableResponse) => void
  ): grpcWeb.ClientReadableStream<unlockable_pb.LockUnlockableResponse>;

}

export class UnlockableHistoryClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listUnlockableHistory(
    request: unlockable_pb.ListUnlockableHistoryRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: unlockable_pb.ListUnlockableHistoryResponse) => void
  ): grpcWeb.ClientReadableStream<unlockable_pb.ListUnlockableHistoryResponse>;

}

export class UnlockableAPIPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listUnlockables(
    request: unlockable_pb.ListUnlockablesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<unlockable_pb.ListUnlockablesResponse>;

  unlockUnlockable(
    request: unlockable_pb.UnlockUnlockableRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<unlockable_pb.UnlockUnlockableResponse>;

  lockUnlockable(
    request: unlockable_pb.LockUnlockableRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<unlockable_pb.LockUnlockableResponse>;

}

export class UnlockableHistoryPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  listUnlockableHistory(
    request: unlockable_pb.ListUnlockableHistoryRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<unlockable_pb.ListUnlockableHistoryResponse>;

}

