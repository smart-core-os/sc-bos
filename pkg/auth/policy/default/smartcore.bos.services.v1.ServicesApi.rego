package smartcore.bos.services.v1.ServicesApi

import data.scutil.token.token_has_role
import data.scutil.rpc.verb_match
import data.scutil.rpc.read_request

default allow := false # take over all permissions for this service

# admin based access is unrestricted
allow if token_has_role("admin")
allow if token_has_role("super-admin")
# certificate based access is unrestricted, this may change in future
allow if input.certificate_valid

# Commissioners can do anything with services
allow if token_has_role("commissioner")

# Operators are allowed extra privileges to start/stop any service/automation.
# Also operators can fully manage zones as they see fit.
allow if {
  token_has_role("operator")
  read_request
}
allow if {
  token_has_role("operator")
  verb_match({"Stop", "Start"})
}
allow if {
  token_has_role("operator")
  endswith(input.request.name, "/zones")
}
allow if {
  token_has_role("operator")
  input.request.name == "zones"
}

allow if {
  token_has_role("viewer")
  read_request
}

# signage has no rights here

# Allow anyone to get service metadata about any service.
allow if input.method == "GetServiceMetadata"
allow if input.method == "PullServiceMetadata"
