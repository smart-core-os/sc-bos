package scutil.token

roles := {
  "admin", # unrestricted user access
  "commissioner",
  "operator",
  "signage",
  "super-admin",
  "viewer"
}

valid_claims := claims if {
  input.token_valid
  claims := input.token_claims
}

token_roles := roles if {
  claims := valid_claims
  roles := claims.roles
}

token_permission_assignments := assignments if {
  claims := valid_claims
  assignments := claims.permissions
}

token_has_role(role) if {
  role in roles
  claims := valid_claims
  claims.system_roles[_] == role
}

token_has_permission_direct(permission) if {
  some assignment in token_permission_assignments
  assignment.permission == permission
  not assignment.scoped
}

token_has_permission_direct(permission) if {
  some assignment in token_permission_assignments
  assignment.permission == permission
  assignment.scoped
  assignment.resource_type == "NAMED_RESOURCE_PATH_PREFIX"
  startswith(input.request.name, concat("/", [assignment.resource, ""]))
}

token_has_permission_direct(permission) if {
  some assignment in token_permission_assignments
  assignment.permission == permission
  assignment.scoped
  assignment.resource_type in ["NAMED_RESOURCE", "NAMED_RESOURCE_PATH_PREFIX"]
  input.request.name == assignment.resource
}

token_has_permission(permission) if {
  token_has_permission_direct(permission)
}

token_has_permission(permission) if {
  some parent_permission
  permission in inherit_permissions[parent_permission]
  token_has_permission_direct(parent_permission)
}

inherit_permissions := {
  "trait:*": {"trait:read", "trait:write"},
  "trait:write": {"trait:read"},
}
