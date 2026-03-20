import * as jspb from 'google-protobuf'



export class PageToken extends jspb.Message {
  getLastOffset(): number;
  setLastOffset(value: number): PageToken;
  hasLastOffset(): boolean;
  clearLastOffset(): PageToken;

  getLastResourceName(): string;
  setLastResourceName(value: string): PageToken;
  hasLastResourceName(): boolean;
  clearLastResourceName(): PageToken;

  getPageStartCase(): PageToken.PageStartCase;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PageToken.AsObject;
  static toObject(includeInstance: boolean, msg: PageToken): PageToken.AsObject;
  static serializeBinaryToWriter(message: PageToken, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PageToken;
  static deserializeBinaryFromReader(message: PageToken, reader: jspb.BinaryReader): PageToken;
}

export namespace PageToken {
  export type AsObject = {
    lastOffset?: number;
    lastResourceName?: string;
  };

  export enum PageStartCase {
    PAGE_START_NOT_SET = 0,
    LAST_OFFSET = 1,
    LAST_RESOURCE_NAME = 2,
  }
}

