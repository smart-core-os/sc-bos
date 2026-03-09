import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_log_v1_log_pb from '../../../../smartcore/bos/log/v1/log_pb'; // proto import: "smartcore/bos/log/v1/log.proto"


export class LogApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  pullLogMessages(
    request: smartcore_bos_log_v1_log_pb.PullLogMessagesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_log_v1_log_pb.PullLogMessagesResponse>;

  getLogLevel(
    request: smartcore_bos_log_v1_log_pb.GetLogLevelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_log_v1_log_pb.LogLevel) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_log_v1_log_pb.LogLevel>;

  pullLogLevel(
    request: smartcore_bos_log_v1_log_pb.PullLogLevelRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_log_v1_log_pb.PullLogLevelResponse>;

  updateLogLevel(
    request: smartcore_bos_log_v1_log_pb.UpdateLogLevelRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_log_v1_log_pb.LogLevel) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_log_v1_log_pb.LogLevel>;

  getLogMetadata(
    request: smartcore_bos_log_v1_log_pb.GetLogMetadataRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_log_v1_log_pb.LogMetadata) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_log_v1_log_pb.LogMetadata>;

  pullLogMetadata(
    request: smartcore_bos_log_v1_log_pb.PullLogMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_log_v1_log_pb.PullLogMetadataResponse>;

  getDownloadLogUrl(
    request: smartcore_bos_log_v1_log_pb.GetDownloadLogUrlRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_log_v1_log_pb.GetDownloadLogUrlResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_log_v1_log_pb.GetDownloadLogUrlResponse>;

}

export class LogApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  pullLogMessages(
    request: smartcore_bos_log_v1_log_pb.PullLogMessagesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_log_v1_log_pb.PullLogMessagesResponse>;

  getLogLevel(
    request: smartcore_bos_log_v1_log_pb.GetLogLevelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_log_v1_log_pb.LogLevel>;

  pullLogLevel(
    request: smartcore_bos_log_v1_log_pb.PullLogLevelRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_log_v1_log_pb.PullLogLevelResponse>;

  updateLogLevel(
    request: smartcore_bos_log_v1_log_pb.UpdateLogLevelRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_log_v1_log_pb.LogLevel>;

  getLogMetadata(
    request: smartcore_bos_log_v1_log_pb.GetLogMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_log_v1_log_pb.LogMetadata>;

  pullLogMetadata(
    request: smartcore_bos_log_v1_log_pb.PullLogMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_log_v1_log_pb.PullLogMetadataResponse>;

  getDownloadLogUrl(
    request: smartcore_bos_log_v1_log_pb.GetDownloadLogUrlRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_log_v1_log_pb.GetDownloadLogUrlResponse>;

}

