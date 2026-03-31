import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_metadata_v1_metadata_pb from '../../../../smartcore/bos/metadata/v1/metadata_pb'; // proto import: "smartcore/bos/metadata/v1/metadata.proto"


export class MetadataApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getMetadata(
    request: smartcore_bos_metadata_v1_metadata_pb.GetMetadataRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_metadata_v1_metadata_pb.Metadata) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_metadata_v1_metadata_pb.Metadata>;

  pullMetadata(
    request: smartcore_bos_metadata_v1_metadata_pb.PullMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_metadata_v1_metadata_pb.PullMetadataResponse>;

}

export class MetadataInfoClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

export class MetadataApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getMetadata(
    request: smartcore_bos_metadata_v1_metadata_pb.GetMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_metadata_v1_metadata_pb.Metadata>;

  pullMetadata(
    request: smartcore_bos_metadata_v1_metadata_pb.PullMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_metadata_v1_metadata_pb.PullMetadataResponse>;

}

export class MetadataInfoPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

}

