import * as grpcWeb from 'grpc-web';

import * as traits_lock_unlock_pb from '../traits/lock_unlock_pb'; // proto import: "traits/lock_unlock.proto"


export class LockUnlockApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getLockUnlock(
    request: traits_lock_unlock_pb.GetLockUnlockRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_lock_unlock_pb.LockUnlock) => void
  ): grpcWeb.ClientReadableStream<traits_lock_unlock_pb.LockUnlock>;

  updateLockUnlock(
    request: traits_lock_unlock_pb.UpdateLockUnlockRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_lock_unlock_pb.LockUnlock) => void
  ): grpcWeb.ClientReadableStream<traits_lock_unlock_pb.LockUnlock>;

  pullLockUnlock(
    request: traits_lock_unlock_pb.PullLockUnlockRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_lock_unlock_pb.PullLockUnlockResponse>;

}

export class LockUnlockInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

export class LockUnlockApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getLockUnlock(
    request: traits_lock_unlock_pb.GetLockUnlockRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_lock_unlock_pb.LockUnlock>;

  updateLockUnlock(
    request: traits_lock_unlock_pb.UpdateLockUnlockRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_lock_unlock_pb.LockUnlock>;

  pullLockUnlock(
    request: traits_lock_unlock_pb.PullLockUnlockRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_lock_unlock_pb.PullLockUnlockResponse>;

}

export class LockUnlockInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

