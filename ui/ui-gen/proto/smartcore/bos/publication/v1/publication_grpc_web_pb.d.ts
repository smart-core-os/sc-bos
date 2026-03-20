import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_publication_v1_publication_pb from '../../../../smartcore/bos/publication/v1/publication_pb'; // proto import: "smartcore/bos/publication/v1/publication.proto"


export class PublicationApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createPublication(
    request: smartcore_bos_publication_v1_publication_pb.CreatePublicationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_publication_v1_publication_pb.Publication) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_publication_v1_publication_pb.Publication>;

  getPublication(
    request: smartcore_bos_publication_v1_publication_pb.GetPublicationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_publication_v1_publication_pb.Publication) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_publication_v1_publication_pb.Publication>;

  updatePublication(
    request: smartcore_bos_publication_v1_publication_pb.UpdatePublicationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_publication_v1_publication_pb.Publication) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_publication_v1_publication_pb.Publication>;

  deletePublication(
    request: smartcore_bos_publication_v1_publication_pb.DeletePublicationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_publication_v1_publication_pb.Publication) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_publication_v1_publication_pb.Publication>;

  pullPublication(
    request: smartcore_bos_publication_v1_publication_pb.PullPublicationRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_publication_v1_publication_pb.PullPublicationResponse>;

  listPublications(
    request: smartcore_bos_publication_v1_publication_pb.ListPublicationsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_publication_v1_publication_pb.ListPublicationsResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_publication_v1_publication_pb.ListPublicationsResponse>;

  pullPublications(
    request: smartcore_bos_publication_v1_publication_pb.PullPublicationsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_publication_v1_publication_pb.PullPublicationsResponse>;

  acknowledgePublication(
    request: smartcore_bos_publication_v1_publication_pb.AcknowledgePublicationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_publication_v1_publication_pb.Publication) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_publication_v1_publication_pb.Publication>;

}

export class PublicationApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createPublication(
    request: smartcore_bos_publication_v1_publication_pb.CreatePublicationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_publication_v1_publication_pb.Publication>;

  getPublication(
    request: smartcore_bos_publication_v1_publication_pb.GetPublicationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_publication_v1_publication_pb.Publication>;

  updatePublication(
    request: smartcore_bos_publication_v1_publication_pb.UpdatePublicationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_publication_v1_publication_pb.Publication>;

  deletePublication(
    request: smartcore_bos_publication_v1_publication_pb.DeletePublicationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_publication_v1_publication_pb.Publication>;

  pullPublication(
    request: smartcore_bos_publication_v1_publication_pb.PullPublicationRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_publication_v1_publication_pb.PullPublicationResponse>;

  listPublications(
    request: smartcore_bos_publication_v1_publication_pb.ListPublicationsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_publication_v1_publication_pb.ListPublicationsResponse>;

  pullPublications(
    request: smartcore_bos_publication_v1_publication_pb.PullPublicationsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_publication_v1_publication_pb.PullPublicationsResponse>;

  acknowledgePublication(
    request: smartcore_bos_publication_v1_publication_pb.AcknowledgePublicationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_publication_v1_publication_pb.Publication>;

}

