package scos

user_request(service, method, request, roles) := result if {
  result := {
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

# --- Old-style traits (pre-migration names) ---

test_viewer_GetBrightness if {
  data.smartcore.allow with input as user_request("smartcore.traits.LightApi", "GetBrightness", {}, ["viewer"])
}
test_viewer_UpdateBrightness if {
  not data.smartcore.traits.LightApi.allow with input as user_request("smartcore.traits.LightApi", "UpdateBrightness", {}, ["viewer"])
  not data.smartcore.traits.allow with input as user_request("smartcore.traits.LightApi", "UpdateBrightness", {}, ["viewer"])
  not data.smartcore.allow with input as user_request("smartcore.traits.LightApi", "UpdateBrightness", {}, ["viewer"])
  not data.grpc_default.allow with input as user_request("smartcore.traits.LightApi", "UpdateBrightness", {}, ["viewer"])
}

test_operator_GetBrightness if {
  data.smartcore.allow with input as user_request("smartcore.traits.LightApi", "GetBrightness", {}, ["operator"])
}
test_operator_UpdateBrightness if {
  data.smartcore.allow with input as user_request("smartcore.traits.LightApi", "UpdateBrightness", {}, ["operator"])
}
test_operator_StartService if {
  data.smartcore.bos.services.v1.ServicesApi.allow with input as user_request("smartcore.bos.services.v1.ServicesApi", "StartService", {}, ["operator"])
}
test_operator_ConfigureService if {
  not data.smartcore.bos.services.v1.ServicesApi.allow with input as user_request("smartcore.bos.services.v1.ServicesApi", "ConfigureService", {}, ["operator"])
  not data.smartcore.bos.allow with input as user_request("smartcore.bos.services.v1.ServicesApi", "ConfigureService", {}, ["operator"])
  not data.smartcore.allow with input as user_request("smartcore.bos.services.v1.ServicesApi", "ConfigureService", {}, ["operator"])
  not data.grpc_default.allow with input as user_request("smartcore.bos.services.v1.ServicesApi", "ConfigureService", {}, ["operator"])
}
test_operator_ConfigureService_zones if {
  req := user_request("smartcore.bos.services.v1.ServicesApi", "ConfigureService", {"name": "zones"}, ["operator"])
  data.smartcore.bos.services.v1.ServicesApi.allow with input as req
}
test_operator_ConfigureService_zones if {
  req := user_request("smartcore.bos.services.v1.ServicesApi", "ConfigureService", {"name": "ns/1/zones"}, ["operator"])
  data.smartcore.bos.services.v1.ServicesApi.allow with input as req
}
test_operator_DaliApi if {
  data.smartcore.bos.driver.dali.v1.DaliApi.allow with input as user_request("smartcore.bos.driver.dali.v1.DaliApi", "StartTest", {}, ["operator"])
  data.smartcore.bos.driver.dali.v1.DaliApi.allow with input as user_request("smartcore.bos.driver.dali.v1.DaliApi", "StopTest", {}, ["operator"])
  not data.smartcore.bos.driver.dali.v1.DaliApi.allow with input as user_request("smartcore.bos.driver.dali.v1.DaliApi", "DeleteTestResult", {}, ["operator"])
}

tenant_request(service, method, request, zones) := result if {
  result := {
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

test_zone_exact if {
  data.smartcore.traits.allow with input as tenant_request("smartcore.traits.LightApi", "GetBrightness", {"name": "zone/1"}, ["zone/1"])
}
test_zone_parent if {
  data.smartcore.traits.allow with input as tenant_request("smartcore.traits.LightApi", "GetBrightness", {"name": "zone/1/child"}, ["zone/1"])
}
test_zone_mismatch if {
  not data.smartcore.traits.LightApi.allow with input as tenant_request("smartcore.traits.LightApi", "GetBrightness", {"name": "zone/2"}, ["zone/1"])
  not data.smartcore.traits.allow with input as tenant_request("smartcore.traits.LightApi", "GetBrightness", {"name": "zone/2"}, ["zone/1"])
  not data.smartcore.allow with input as tenant_request("smartcore.traits.LightApi", "GetBrightness", {"name": "zone/2"}, ["zone/1"])
  not data.grpc_default.allow with input as tenant_request("smartcore.traits.LightApi", "GetBrightness", {"name": "zone/2"}, ["zone/1"])
}

# --- New-style bos trait service (post-migration names) ---

test_bos_viewer_GetBrightness if {
  data.smartcore.allow with input as user_request("smartcore.bos.light.v1.LightApi", "GetBrightness", {}, ["viewer"])
}
test_bos_viewer_UpdateBrightness if {
  not data.smartcore.bos.light.v1.LightApi.allow with input as user_request("smartcore.bos.light.v1.LightApi", "UpdateBrightness", {}, ["viewer"])
  not data.smartcore.bos.allow with input as user_request("smartcore.bos.light.v1.LightApi", "UpdateBrightness", {}, ["viewer"])
  not data.smartcore.allow with input as user_request("smartcore.bos.light.v1.LightApi", "UpdateBrightness", {}, ["viewer"])
  not data.grpc_default.allow with input as user_request("smartcore.bos.light.v1.LightApi", "UpdateBrightness", {}, ["viewer"])
}
test_bos_operator_GetBrightness if {
  data.smartcore.allow with input as user_request("smartcore.bos.light.v1.LightApi", "GetBrightness", {}, ["operator"])
}
test_bos_operator_UpdateBrightness if {
  data.smartcore.allow with input as user_request("smartcore.bos.light.v1.LightApi", "UpdateBrightness", {}, ["operator"])
}

# Tenant permission-based access via smartcore.bos.
# known_traits is mocked here because it is normally injected at runtime via data.system;
# standalone `opa test` does not have that data available.
mock_known_traits := [{"grpc_services": ["smartcore.bos.light.v1.LightApi"]}]

test_bos_zone_exact if {
  data.smartcore.bos.allow
    with input as tenant_request("smartcore.bos.light.v1.LightApi", "GetBrightness", {"name": "zone/1"}, ["zone/1"])
    with data.system.known_traits as mock_known_traits
}
test_bos_zone_parent if {
  data.smartcore.bos.allow
    with input as tenant_request("smartcore.bos.light.v1.LightApi", "GetBrightness", {"name": "zone/1/child"}, ["zone/1"])
    with data.system.known_traits as mock_known_traits
}
test_bos_zone_mismatch if {
  not data.smartcore.bos.light.v1.LightApi.allow
    with input as tenant_request("smartcore.bos.light.v1.LightApi", "GetBrightness", {"name": "zone/2"}, ["zone/1"])
    with data.system.known_traits as mock_known_traits
  not data.smartcore.bos.allow
    with input as tenant_request("smartcore.bos.light.v1.LightApi", "GetBrightness", {"name": "zone/2"}, ["zone/1"])
    with data.system.known_traits as mock_known_traits
  not data.smartcore.allow
    with input as tenant_request("smartcore.bos.light.v1.LightApi", "GetBrightness", {"name": "zone/2"}, ["zone/1"])
    with data.system.known_traits as mock_known_traits
  not data.grpc_default.allow
    with input as tenant_request("smartcore.bos.light.v1.LightApi", "GetBrightness", {"name": "zone/2"}, ["zone/1"])
    with data.system.known_traits as mock_known_traits
}

# --- CloudConnectionApi: mutations restricted to admin / cert; reads gated on trait:read ---

cert_request(service, method, request) := result if {
  result := {
    "service": service,
    "method": method,
    "stream": {"is_server_stream": false, "is_client_stream": false, "open": false},
    "request": request,
    "certificate_present": true,
    "certificate_valid": true,
    "certificate": {},
    "token_present": false,
    "token_valid": false,
    "token_claims": null
  }
}

permission_request(service, method, request, permissions) := result if {
  result := {
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
      "system_roles": [],
      "is_service": true,
      "permissions": [ {"permission": p, "scoped": false} | p := permissions[_] ]
    }
  }
}

cloud_service := "smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi"

test_cloud_admin_Register if {
  data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as user_request(cloud_service, "RegisterCloudConnection", {}, ["admin"])
}
test_cloud_super_admin_Unlink if {
  data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as user_request(cloud_service, "UnlinkCloudConnection", {}, ["super-admin"])
}
test_cloud_cert_Register if {
  data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as cert_request(cloud_service, "RegisterCloudConnection", {})
}
test_cloud_cert_Unlink if {
  data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as cert_request(cloud_service, "UnlinkCloudConnection", {})
}

# trait:write must NOT be sufficient for mutating cloud connection ops —
# only admin role and cert auth can register/unlink.
test_cloud_trait_write_Register_denied if {
  not data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as permission_request(cloud_service, "RegisterCloudConnection", {}, ["trait:write"])
}
test_cloud_trait_star_Unlink_denied if {
  not data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as permission_request(cloud_service, "UnlinkCloudConnection", {}, ["trait:*"])
}
# Legacy named roles do not grant mutating access either.
# Commissioners get unrestricted access via smartcore.allow, but the
# service-level `default allow := false` short-circuits the hierarchy.
test_cloud_commissioner_Register_denied if {
  not data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as user_request(cloud_service, "RegisterCloudConnection", {}, ["commissioner"])
}
test_cloud_operator_Unlink_denied if {
  not data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as user_request(cloud_service, "UnlinkCloudConnection", {}, ["operator"])
}

# Read ops are gated on trait:read; trait:write satisfies this via inheritance.
test_cloud_trait_read_Get if {
  data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as permission_request(cloud_service, "GetCloudConnection", {}, ["trait:read"])
}
test_cloud_trait_read_Pull if {
  data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as permission_request(cloud_service, "PullCloudConnection", {}, ["trait:read"])
}
test_cloud_trait_read_GetDefaults if {
  data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as permission_request(cloud_service, "GetCloudConnectionDefaults", {}, ["trait:read"])
}
test_cloud_trait_write_Get if {
  data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as permission_request(cloud_service, "GetCloudConnection", {}, ["trait:write"])
}

# Test op is gated on trait:write — trait:read is insufficient.
test_cloud_trait_write_Test if {
  data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as permission_request(cloud_service, "TestCloudConnection", {}, ["trait:write"])
}
test_cloud_trait_read_Test_denied if {
  not data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as permission_request(cloud_service, "TestCloudConnection", {}, ["trait:read"])
}

# Tokens with no relevant permissions can do nothing here.
test_cloud_no_perms_Get_denied if {
  not data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as permission_request(cloud_service, "GetCloudConnection", {}, [])
}
test_cloud_no_perms_Test_denied if {
  not data.smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi.allow
    with input as permission_request(cloud_service, "TestCloudConnection", {}, [])
}
