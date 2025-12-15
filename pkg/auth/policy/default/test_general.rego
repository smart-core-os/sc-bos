package scos

import future.keywords.in

user_request(service, method, request, roles) := input {
  input := {
    "service": service,
    "method": method,
    "stream": {"is_server_stream": false, "is_client_stream": false, "open": false},
    "request": request,
    "certificate_present": false,
    "certificate_valid": false,
    "certificate": null,
    "token_present": true,
    "token_valid": true,
    "token_claims": {
      "system_roles": roles,
      "is_service": true
    }
  }
}

test_viewer_GetBrightness {
  data.smartcore.allow with input as user_request("smartcore.traits.LightApi", "GetBrightness", {}, ["viewer"])
}
test_viewer_UpdateBrightness {
  not data.smartcore.traits.LightApi.allow with input as user_request("smartcore.traits.LightApi", "UpdateBrightness", {}, ["viewer"])
  not data.smartcore.traits.allow with input as user_request("smartcore.traits.LightApi", "UpdateBrightness", {}, ["viewer"])
  not data.smartcore.allow with input as user_request("smartcore.traits.LightApi", "UpdateBrightness", {}, ["viewer"])
  not data.grpc_default.allow with input as user_request("smartcore.traits.LightApi", "UpdateBrightness", {}, ["viewer"])
}

test_operator_GetBrightness {
  data.smartcore.allow with input as user_request("smartcore.traits.LightApi", "GetBrightness", {}, ["operator"])
}
test_operator_UpdateBrightness {
  data.smartcore.allow with input as user_request("smartcore.traits.LightApi", "UpdateBrightness", {}, ["operator"])
}
test_operator_StartService {
  data.smartcore.bos.ServicesApi.allow with input as user_request("smartcore.bos.services.v1.ServicesApi", "StartService", {}, ["operator"])
}
test_operator_ConfigureService {
  not data.smartcore.bos.services.v1.ServicesApi.allow with input as user_request("smartcore.bos.services.v1.ServicesApi", "ConfigureService", {}, ["operator"])
  not data.smartcore.bos.allow with input as user_request("smartcore.bos.services.v1.ServicesApi", "ConfigureService", {}, ["operator"])
  not data.smartcore.allow with input as user_request("smartcore.bos.services.v1.ServicesApi", "ConfigureService", {}, ["operator"])
  not data.grpc_default.allow with input as user_request("smartcore.bos.services.v1.ServicesApi", "ConfigureService", {}, ["operator"])
}
test_operator_ConfigureService_zones {
  input := user_request("smartcore.bos.services.v1.ServicesApi", "ConfigureService", {
    "name": "zones"
  }, ["operator"])
  data.smartcore.bos.services.v1.ServicesApi.allow with input as input
}
test_operator_ConfigureService_zones {
  input := user_request("smartcore.bos.services.v1.ServicesApi", "ConfigureService", {
    "name": "ns/1/zones"
  }, ["operator"])
  data.smartcore.bos.services.v1.ServicesApi.allow with input as input
}
test_operator_DaliApi {
  data.smartcore.bos.driver.dali.DaliApi.allow with input as user_request("smartcore.bos.driver.dali.DaliApi", "StartTest", {}, ["operator"])
  data.smartcore.bos.driver.dali.DaliApi.allow with input as user_request("smartcore.bos.driver.dali.DaliApi", "StopTest", {}, ["operator"])
  not data.smartcore.bos.driver.dali.DaliApi.allow with input as user_request("smartcore.bos.driver.dali.DaliApi", "DeleteTestResult", {}, ["operator"])
}

tenant_request(service, method, request, zones) := input {
  input := {
    "service": service,
    "method": method,
    "stream": {"is_server_stream": false, "is_client_stream": false, "open": false},
    "request": request,
    "certificate_present": false,
    "certificate_valid": false,
    "certificate": null,
    "token_present": true,
    "token_valid": true,
    "token_claims": {
      "system_roles": null,
      "is_service": true,
      "permissions": [ permission |
        zone := zones[_]
        permission := {
          "permission": "trait:*",
          "scoped": true,
          "resource_type": "NAMED_RESOURCE_PATH_PREFIX",
          "resource": zone
        }
      ],
    }
  }
}

test_zone_exact {
  data.smartcore.traits.allow with input as tenant_request("smartcore.traits.LightApi", "GetBrightness", {"name": "zone/1"}, ["zone/1"])
}
test_zone_parent {
  data.smartcore.traits.allow with input as tenant_request("smartcore.traits.LightApi", "GetBrightness", {"name": "zone/1/child"}, ["zone/1"])
}
test_zone_mismatch {
  not data.smartcore.traits.LightApi.allow with input as tenant_request("smartcore.traits.LightApi", "GetBrightness", {"name": "zone/2"}, ["zone/1"])
  not data.smartcore.traits.allow with input as tenant_request("smartcore.traits.LightApi", "GetBrightness", {"name": "zone/2"}, ["zone/1"])
  not data.smartcore.allow with input as tenant_request("smartcore.traits.LightApi", "GetBrightness", {"name": "zone/2"}, ["zone/1"])
  not data.grpc_default.allow with input as tenant_request("smartcore.traits.LightApi", "GetBrightness", {"name": "zone/2"}, ["zone/1"])
}
