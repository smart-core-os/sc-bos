import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_lockunlock_v1_lock_unlock_pb from '../../../../smartcore/bos/lockunlock/v1/lock_unlock_pb'; // proto import: "smartcore/bos/lockunlock/v1/lock_unlock.proto"


export class LockUnlockApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getLockUnlock(
    request: smartcore_bos_lockunlock_v1_lock_unlock_pb.GetLockUnlockRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_lockunlock_v1_lock_unlock_pb.LockUnlock) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_lockunlock_v1_lock_unlock_pb.LockUnlock>;

  updateLockUnlock(
    request: smartcore_bos_lockunlock_v1_lock_unlock_pb.UpdateLockUnlockRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_lockunlock_v1_lock_unlock_pb.LockUnlock) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_lockunlock_v1_lock_unlock_pb.LockUnlock>;

  pullLockUnlock(
    request: smartcore_bos_lockunlock_v1_lock_unlock_pb.PullLockUnlockRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_lockunlock_v1_lock_unlock_pb.PullLockUnlockResponse>;

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
    request: smartcore_bos_lockunlock_v1_lock_unlock_pb.GetLockUnlockRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_lockunlock_v1_lock_unlock_pb.LockUnlock>;

  updateLockUnlock(
    request: smartcore_bos_lockunlock_v1_lock_unlock_pb.UpdateLockUnlockRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_lockunlock_v1_lock_unlock_pb.LockUnlock>;

  pullLockUnlock(
    request: smartcore_bos_lockunlock_v1_lock_unlock_pb.PullLockUnlockRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_lockunlock_v1_lock_unlock_pb.PullLockUnlockResponse>;

}

export class LockUnlockInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

