package smartcore.traits

import data.scutil.token.token_has_permission
import data.scutil.rpc.read_request

allow if {
  token_has_permission("trait:read")
  read_request
}

allow if token_has_permission("trait:write")
