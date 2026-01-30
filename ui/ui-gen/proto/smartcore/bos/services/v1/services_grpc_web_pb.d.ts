import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_services_v1_services_pb from '../../../../smartcore/bos/services/v1/services_pb'; // proto import: "smartcore/bos/services/v1/services.proto"


export class ServicesApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getService(
    request: smartcore_bos_services_v1_services_pb.GetServiceRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_services_v1_services_pb.Service) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_services_v1_services_pb.Service>;

  pullService(
    request: smartcore_bos_services_v1_services_pb.PullServiceRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_services_v1_services_pb.PullServiceResponse>;

  createService(
    request: smartcore_bos_services_v1_services_pb.CreateServiceRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_services_v1_services_pb.Service) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_services_v1_services_pb.Service>;

  deleteService(
    request: smartcore_bos_services_v1_services_pb.DeleteServiceRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_services_v1_services_pb.Service) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_services_v1_services_pb.Service>;

  listServices(
    request: smartcore_bos_services_v1_services_pb.ListServicesRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_services_v1_services_pb.ListServicesResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_services_v1_services_pb.ListServicesResponse>;

  pullServices(
    request: smartcore_bos_services_v1_services_pb.PullServicesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_services_v1_services_pb.PullServicesResponse>;

  startService(
    request: smartcore_bos_services_v1_services_pb.StartServiceRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_services_v1_services_pb.Service) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_services_v1_services_pb.Service>;

  configureService(
    request: smartcore_bos_services_v1_services_pb.ConfigureServiceRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_services_v1_services_pb.Service) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_services_v1_services_pb.Service>;

  stopService(
    request: smartcore_bos_services_v1_services_pb.StopServiceRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_services_v1_services_pb.Service) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_services_v1_services_pb.Service>;

  getServiceMetadata(
    request: smartcore_bos_services_v1_services_pb.GetServiceMetadataRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_services_v1_services_pb.ServiceMetadata) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_services_v1_services_pb.ServiceMetadata>;

  pullServiceMetadata(
    request: smartcore_bos_services_v1_services_pb.PullServiceMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_services_v1_services_pb.PullServiceMetadataResponse>;

}

export class ServicesApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getService(
    request: smartcore_bos_services_v1_services_pb.GetServiceRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_services_v1_services_pb.Service>;

  pullService(
    request: smartcore_bos_services_v1_services_pb.PullServiceRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_services_v1_services_pb.PullServiceResponse>;

  createService(
    request: smartcore_bos_services_v1_services_pb.CreateServiceRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_services_v1_services_pb.Service>;

  deleteService(
    request: smartcore_bos_services_v1_services_pb.DeleteServiceRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_services_v1_services_pb.Service>;

  listServices(
    request: smartcore_bos_services_v1_services_pb.ListServicesRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_services_v1_services_pb.ListServicesResponse>;

  pullServices(
    request: smartcore_bos_services_v1_services_pb.PullServicesRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_services_v1_services_pb.PullServicesResponse>;

  startService(
    request: smartcore_bos_services_v1_services_pb.StartServiceRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_services_v1_services_pb.Service>;

  configureService(
    request: smartcore_bos_services_v1_services_pb.ConfigureServiceRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_services_v1_services_pb.Service>;

  stopService(
    request: smartcore_bos_services_v1_services_pb.StopServiceRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_services_v1_services_pb.Service>;

  getServiceMetadata(
    request: smartcore_bos_services_v1_services_pb.GetServiceMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_services_v1_services_pb.ServiceMetadata>;

  pullServiceMetadata(
    request: smartcore_bos_services_v1_services_pb.PullServiceMetadataRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<smartcore_bos_services_v1_services_pb.PullServiceMetadataResponse>;

}

