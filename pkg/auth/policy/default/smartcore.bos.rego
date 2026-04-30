package smartcore.bos

import data.scutil.rpc.read_request
import data.scutil.token.token_has_permission
import data.system.known_traits

trait_request if {
  some trait in known_traits
  input.service in trait.grpc_services
}

allow if {
  trait_request
  read_request
  token_has_permission("trait:read")
}

allow if {
  trait_request
  token_has_permission("trait:write")
}
