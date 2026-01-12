import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_driver_axiomxa_v1_axiomxa_pb from '../../../../../smartcore/bos/driver/axiomxa/v1/axiomxa_pb'; // proto import: "smartcore/bos/driver/axiomxa/v1/axiomxa.proto"


export class AxiomXaDriverServiceClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  saveQRCredential(
    request: smartcore_bos_driver_axiomxa_v1_axiomxa_pb.SaveQRCredentialRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_driver_axiomxa_v1_axiomxa_pb.SaveQRCredentialResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_driver_axiomxa_v1_axiomxa_pb.SaveQRCredentialResponse>;

}

export class AxiomXaDriverServicePromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  saveQRCredential(
    request: smartcore_bos_driver_axiomxa_v1_axiomxa_pb.SaveQRCredentialRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_driver_axiomxa_v1_axiomxa_pb.SaveQRCredentialResponse>;

}

