package smartcore.bos.dataretention.v1.DataRetentionApi

import data.scutil.rpc.read_request
import data.scutil.token.token_has_role
import data.scutil.token.token_has_permission

default allow := false

# Unrestricted access for admin roles and valid certificates.
allow { token_has_role("admin") }
allow { token_has_role("super-admin") }
allow { input.certificate_valid }
allow { token_has_role("commissioner") }

# Operators and viewers may read data retention statistics.
allow {
  token_has_role("operator")
  read_request
}
allow {
  token_has_role("viewer")
  read_request
}

# Tenant permission-based read access.
allow {
  token_has_permission("trait:read")
  read_request
}
