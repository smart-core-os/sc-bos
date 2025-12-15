package smartcore.bos

import future.keywords.in

import data.scutil.rpc.read_request
import data.scutil.token.token_has_permission
import data.system.known_traits

trait_request {
  some trait in known_traits
  input.service in trait.grpc_services
}

allow {
  trait_request
  read_request
  token_has_permission("trait:read")
}

allow {
  trait_request
  token_has_permission("trait:write")
}
