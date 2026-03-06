// source: types/connection.proto
/**
 * @fileoverview
 * @enhanceable
 * @suppress {missingRequire} reports error on implicit type usages.
 * @suppress {messageConventions} JS Compiler reports an error if a variable or
 *     field starts with 'MSG_' and isn't a translatable message.
 * @public
 */
// GENERATED CODE -- DO NOT EDIT!
/* eslint-disable */
// @ts-nocheck

var jspb = require('google-protobuf');
var goog = jspb;
var global = globalThis;

goog.exportSymbol('proto.smartcore.types.CommStatus', null, global);
goog.exportSymbol('proto.smartcore.types.Connectivity', null, global);
/**
 * @enum {number}
 */
proto.smartcore.types.Connectivity = {
  CONNECTIVITY_UNSPECIFIED: 0,
  NOT_APPLICABLE: 1,
  DISCONNECTED: 2,
  CONNECTED: 3
};

/**
 * @enum {number}
 */
proto.smartcore.types.CommStatus = {
  COMM_STATUS_UNSPECIFIED: 0,
  COMM_SUCCESS: 1,
  COMM_FAILURE: 2
};

goog.object.extend(exports, proto.smartcore.types);
