package smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi

import data.scutil.rpc.read_request
import data.scutil.token.token_has_permission
import data.scutil.token.token_has_role

default allow := false

# Admin tokens have unrestricted access.
allow if token_has_role("admin")
allow if token_has_role("super-admin")
# Certificate-authenticated callers (e.g. other cohort nodes) have unrestricted access.
allow if input.certificate_valid

allow if {
  read_request
  token_has_permission("trait:read")
}

# TestCloudConnection is not configuration - regular write access is sufficient
allow if {
  input.method == "TestCloudConnection"
  token_has_permission("trait:write")
}
