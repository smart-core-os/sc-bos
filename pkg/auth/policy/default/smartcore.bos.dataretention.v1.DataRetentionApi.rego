package smartcore.bos.dataretention.v1.DataRetentionApi

import data.scutil.rpc.read_request
import data.scutil.token.token_has_role
import data.scutil.token.token_has_permission

default allow := false

# Unrestricted access for admin roles and valid certificates.
allow if token_has_role("admin")
allow if token_has_role("super-admin")
allow if input.certificate_valid
allow if token_has_role("commissioner")

# Operators and viewers may read data retention statistics.
allow if {
	token_has_role("operator")
	read_request
}
allow if {
	token_has_role("viewer")
	read_request
}

# Tenant permission-based read access.
allow if {
	token_has_permission("trait:read")
	read_request
}
