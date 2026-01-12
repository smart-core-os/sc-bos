import * as grpcWeb from 'grpc-web';

import * as smartcore_bos_enrollment_v1_enrollment_pb from '../../../../smartcore/bos/enrollment/v1/enrollment_pb'; // proto import: "smartcore/bos/enrollment/v1/enrollment.proto"


export class EnrollmentApiClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getEnrollment(
    request: smartcore_bos_enrollment_v1_enrollment_pb.GetEnrollmentRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_enrollment_v1_enrollment_pb.Enrollment) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_enrollment_v1_enrollment_pb.Enrollment>;

  createEnrollment(
    request: smartcore_bos_enrollment_v1_enrollment_pb.CreateEnrollmentRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_enrollment_v1_enrollment_pb.Enrollment) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_enrollment_v1_enrollment_pb.Enrollment>;

  updateEnrollment(
    request: smartcore_bos_enrollment_v1_enrollment_pb.UpdateEnrollmentRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_enrollment_v1_enrollment_pb.Enrollment) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_enrollment_v1_enrollment_pb.Enrollment>;

  deleteEnrollment(
    request: smartcore_bos_enrollment_v1_enrollment_pb.DeleteEnrollmentRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_enrollment_v1_enrollment_pb.Enrollment) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_enrollment_v1_enrollment_pb.Enrollment>;

  testEnrollment(
    request: smartcore_bos_enrollment_v1_enrollment_pb.TestEnrollmentRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.RpcError,
               response: smartcore_bos_enrollment_v1_enrollment_pb.TestEnrollmentResponse) => void
  ): grpcWeb.ClientReadableStream<smartcore_bos_enrollment_v1_enrollment_pb.TestEnrollmentResponse>;

}

export class EnrollmentApiPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getEnrollment(
    request: smartcore_bos_enrollment_v1_enrollment_pb.GetEnrollmentRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_enrollment_v1_enrollment_pb.Enrollment>;

  createEnrollment(
    request: smartcore_bos_enrollment_v1_enrollment_pb.CreateEnrollmentRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_enrollment_v1_enrollment_pb.Enrollment>;

  updateEnrollment(
    request: smartcore_bos_enrollment_v1_enrollment_pb.UpdateEnrollmentRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_enrollment_v1_enrollment_pb.Enrollment>;

  deleteEnrollment(
    request: smartcore_bos_enrollment_v1_enrollment_pb.DeleteEnrollmentRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_enrollment_v1_enrollment_pb.Enrollment>;

  testEnrollment(
    request: smartcore_bos_enrollment_v1_enrollment_pb.TestEnrollmentRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<smartcore_bos_enrollment_v1_enrollment_pb.TestEnrollmentResponse>;

}

