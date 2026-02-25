/**
 * Type information for the well known /__/scos/config.json document used to communication server config to the UI.
 */
export interface ServerConfig {
  grpcAddress: string;
  httpAddress: string;
  httpsAddress: string;
  insecure?: boolean;
  selfSigned?: boolean;
  mutualTls?: boolean;
}
