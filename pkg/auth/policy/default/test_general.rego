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

# --- BootApi: Reboot restricted to admin/cert/commissioner; trait:write is not sufficient ---

boot_service := "smartcore.bos.boot.v1.BootApi"

# Admin, super-admin, cert, and commissioner have unrestricted access (including Reboot).
test_boot_admin_Reboot if {
  data.smartcore.bos.boot.v1.BootApi.allow
    with input as user_request(boot_service, "Reboot", {}, ["admin"])
}
test_boot_super_admin_Reboot if {
  data.smartcore.bos.boot.v1.BootApi.allow
    with input as user_request(boot_service, "Reboot", {}, ["super-admin"])
}
test_boot_cert_Reboot if {
  data.smartcore.bos.boot.v1.BootApi.allow
    with input as cert_request(boot_service, "Reboot", {})
}
test_boot_commissioner_Reboot if {
  data.smartcore.bos.boot.v1.BootApi.allow
    with input as user_request(boot_service, "Reboot", {}, ["commissioner"])
}

# trait:write is NOT sufficient to trigger Reboot — the service-level
# `default allow := false` short-circuits the hierarchy before the blanket
# smartcore.bos.allow trait:write rule is ever reached.
test_boot_trait_write_Reboot_denied if {
  not data.smartcore.bos.boot.v1.BootApi.allow
    with input as permission_request(boot_service, "Reboot", {}, ["trait:write"])
}
test_boot_trait_star_Reboot_denied if {
  not data.smartcore.bos.boot.v1.BootApi.allow
    with input as permission_request(boot_service, "Reboot", {}, ["trait:*"])
}
test_boot_operator_Reboot_denied if {
  not data.smartcore.bos.boot.v1.BootApi.allow
    with input as user_request(boot_service, "Reboot", {}, ["operator"])
}
test_boot_viewer_Reboot_denied if {
  not data.smartcore.bos.boot.v1.BootApi.allow
    with input as user_request(boot_service, "Reboot", {}, ["viewer"])
}

# Operators, viewers, and trait:read can read boot state.
test_boot_operator_GetBootState if {
  data.smartcore.bos.boot.v1.BootApi.allow
    with input as user_request(boot_service, "GetBootState", {}, ["operator"])
}
test_boot_viewer_PullBootState if {
  data.smartcore.bos.boot.v1.BootApi.allow
    with input as user_request(boot_service, "PullBootState", {}, ["viewer"])
}
test_boot_trait_read_GetBootState if {
  data.smartcore.bos.boot.v1.BootApi.allow
    with input as permission_request(boot_service, "GetBootState", {}, ["trait:read"])
}
# trait:write inherits trait:read, so reads are also allowed.
test_boot_trait_write_GetBootState if {
  data.smartcore.bos.boot.v1.BootApi.allow
    with input as permission_request(boot_service, "GetBootState", {}, ["trait:write"])
}

# --- Setup code: the setup role may only enrol the node, read the node's own metadata and use SetupApi ---

enrollment_service := "smartcore.bos.enrollment.v1.EnrollmentApi"
metadata_service := "smartcore.bos.metadata.v1.MetadataApi"
setup_service := "smartcore.bos.setup.v1.SetupApi"

anonymous_request(service, method, request) := result if {
  result := {
    "service": service,
    "method": method,
    "stream": {"is_server_stream": false, "is_client_stream": false, "open": false},
    "request": request,
    "certificate_present": false,
    "certificate_valid": false,
    "certificate": null,
    "token_present": false,
    "token_valid": false,
    "token_claims": null
  }
}

test_setup_CreateEnrollment if {
  data.smartcore.bos.enrollment.v1.EnrollmentApi.allow
    with input as user_request(enrollment_service, "CreateEnrollment", {}, ["setup"])
}
test_anonymous_CreateEnrollment_denied if {
  not data.smartcore.bos.enrollment.v1.EnrollmentApi.allow
    with input as anonymous_request(enrollment_service, "CreateEnrollment", {})
}
test_viewer_CreateEnrollment_denied if {
  not data.smartcore.bos.enrollment.v1.EnrollmentApi.allow
    with input as user_request(enrollment_service, "CreateEnrollment", {}, ["viewer"])
}
# The role only counts when it comes from a valid token.
test_setup_invalid_token_CreateEnrollment_denied if {
  req := object.union(user_request(enrollment_service, "CreateEnrollment", {}, ["setup"]), {"token_valid": false})
  not data.smartcore.bos.enrollment.v1.EnrollmentApi.allow with input as req
}
test_setup_UpdateEnrollment_denied if {
  req := user_request(enrollment_service, "UpdateEnrollment", {}, ["setup"])
  not data.smartcore.bos.enrollment.v1.EnrollmentApi.allow with input as req
  not data.smartcore.bos.allow with input as req
  not data.smartcore.allow with input as req
}
test_setup_DeleteEnrollment_denied if {
  req := user_request(enrollment_service, "DeleteEnrollment", {}, ["setup"])
  not data.smartcore.bos.enrollment.v1.EnrollmentApi.allow with input as req
  not data.smartcore.bos.allow with input as req
  not data.smartcore.allow with input as req
}

test_setup_GetMetadata_node if {
  data.smartcore.bos.metadata.v1.MetadataApi.allow
    with input as user_request(metadata_service, "GetMetadata", {"name": ""}, ["setup"])
}
test_setup_GetMetadata_device_denied if {
  req := user_request(metadata_service, "GetMetadata", {"name": "floor1/light1"}, ["setup"])
  not data.smartcore.bos.metadata.v1.MetadataApi.allow with input as req
  not data.smartcore.bos.allow with input as req
  not data.smartcore.allow with input as req
}
test_setup_PullMetadata_denied if {
  req := user_request(metadata_service, "PullMetadata", {"name": ""}, ["setup"])
  not data.smartcore.bos.metadata.v1.MetadataApi.allow with input as req
  not data.smartcore.bos.allow with input as req
  not data.smartcore.allow with input as req
}
test_anonymous_GetMetadata_node_denied if {
  req := anonymous_request(metadata_service, "GetMetadata", {"name": ""})
  not data.smartcore.bos.metadata.v1.MetadataApi.allow with input as req
  not data.smartcore.bos.allow with input as req
  not data.smartcore.allow with input as req
}
# Other roles keep their general access to metadata.
test_viewer_GetMetadata_device if {
  data.smartcore.allow
    with input as user_request(metadata_service, "GetMetadata", {"name": "floor1/light1"}, ["viewer"])
}

test_setup_trait_denied if {
  req := user_request("smartcore.bos.onoff.v1.OnOffApi", "GetOnOff", {"name": ""}, ["setup"])
  not data.smartcore.bos.allow with input as req
  not data.smartcore.allow with input as req
}
test_setup_CreateAccount_denied if {
  req := user_request("smartcore.bos.account.v1.AccountApi", "CreateAccount", {}, ["setup"])
  not data.smartcore.bos.allow with input as req
  not data.smartcore.allow with input as req
}

test_anonymous_GetSetupState if {
  data.smartcore.bos.setup.v1.SetupApi.allow
    with input as anonymous_request(setup_service, "GetSetupState", {})
}
test_anonymous_CompleteSetup if {
  data.smartcore.bos.setup.v1.SetupApi.allow
    with input as anonymous_request(setup_service, "CompleteSetup", {})
}
test_anonymous_SetupApi_other_denied if {
  not data.smartcore.bos.setup.v1.SetupApi.allow
    with input as anonymous_request(setup_service, "PullSetupState", {})
}

# --- HubApi: a setup code passed on to a node needs the same access as enrolling it ---

hub_service := "smartcore.bos.hub.v1.HubApi"

test_anonymous_InspectHubNode if {
  data.smartcore.bos.hub.v1.HubApi.allow
    with input as anonymous_request(hub_service, "InspectHubNode", {"node": {"address": "ac-01:23557"}, "setupCode": ""})
}
test_anonymous_InspectHubNode_setupCode_denied if {
  req := anonymous_request(hub_service, "InspectHubNode", {"node": {"address": "ac-01:23557"}, "setupCode": "ABCD-EFGH"})
  not data.smartcore.bos.hub.v1.HubApi.allow with input as req
  not data.smartcore.bos.hub.v1.allow with input as req
  not data.smartcore.bos.hub.allow with input as req
  not data.smartcore.bos.allow with input as req
  not data.smartcore.allow with input as req
  not data.grpc_default.allow with input as req
}
test_viewer_InspectHubNode_setupCode_denied if {
  req := user_request(hub_service, "InspectHubNode", {"setupCode": "ABCD-EFGH"}, ["viewer"])
  not data.smartcore.bos.hub.v1.HubApi.allow with input as req
  not data.smartcore.bos.allow with input as req
  not data.smartcore.allow with input as req
}
test_admin_InspectHubNode_setupCode if {
  req := user_request(hub_service, "InspectHubNode", {"setupCode": "ABCD-EFGH"}, ["admin"])
  not data.smartcore.bos.hub.v1.HubApi.allow with input as req
  data.smartcore.allow with input as req
}
# Whoever may enrol a node may also inspect it with its code.
test_commissioner_InspectHubNode_setupCode if {
  data.smartcore.allow with input as user_request(hub_service, "InspectHubNode", {"setupCode": "ABCD-EFGH"}, ["commissioner"])
  data.smartcore.allow with input as user_request(hub_service, "EnrollHubNode", {"setupCode": "ABCD-EFGH"}, ["commissioner"])
}
test_anonymous_EnrollHubNode_denied if {
  req := anonymous_request(hub_service, "EnrollHubNode", {"setupCode": "ABCD-EFGH"})
  not data.smartcore.bos.hub.v1.HubApi.allow with input as req
  not data.smartcore.allow with input as req
}
