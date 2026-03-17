import * as grpcWeb from 'grpc-web';

import * as traits_publication_pb from '../traits/publication_pb'; // proto import: "traits/publication.proto"


export class PublicationApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createPublication(
    request: traits_publication_pb.CreatePublicationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_publication_pb.Publication) => void
  ): grpcWeb.ClientReadableStream<traits_publication_pb.Publication>;

  getPublication(
    request: traits_publication_pb.GetPublicationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_publication_pb.Publication) => void
  ): grpcWeb.ClientReadableStream<traits_publication_pb.Publication>;

  updatePublication(
    request: traits_publication_pb.UpdatePublicationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_publication_pb.Publication) => void
  ): grpcWeb.ClientReadableStream<traits_publication_pb.Publication>;

  deletePublication(
    request: traits_publication_pb.DeletePublicationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_publication_pb.Publication) => void
  ): grpcWeb.ClientReadableStream<traits_publication_pb.Publication>;

  pullPublication(
    request: traits_publication_pb.PullPublicationRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_publication_pb.PullPublicationResponse>;

  listPublications(
    request: traits_publication_pb.ListPublicationsRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_publication_pb.ListPublicationsResponse) => void
  ): grpcWeb.ClientReadableStream<traits_publication_pb.ListPublicationsResponse>;

  pullPublications(
    request: traits_publication_pb.PullPublicationsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_publication_pb.PullPublicationsResponse>;

  acknowledgePublication(
    request: traits_publication_pb.AcknowledgePublicationRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: traits_publication_pb.Publication) => void
  ): grpcWeb.ClientReadableStream<traits_publication_pb.Publication>;

}

export class PublicationApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  createPublication(
    request: traits_publication_pb.CreatePublicationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_publication_pb.Publication>;

  getPublication(
    request: traits_publication_pb.GetPublicationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_publication_pb.Publication>;

  updatePublication(
    request: traits_publication_pb.UpdatePublicationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_publication_pb.Publication>;

  deletePublication(
    request: traits_publication_pb.DeletePublicationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_publication_pb.Publication>;

  pullPublication(
    request: traits_publication_pb.PullPublicationRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_publication_pb.PullPublicationResponse>;

  listPublications(
    request: traits_publication_pb.ListPublicationsRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_publication_pb.ListPublicationsResponse>;

  pullPublications(
    request: traits_publication_pb.PullPublicationsRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<traits_publication_pb.PullPublicationsResponse>;

  acknowledgePublication(
    request: traits_publication_pb.AcknowledgePublicationRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<traits_publication_pb.Publication>;

}

