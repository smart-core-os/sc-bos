package smartcore.bos.ServicesApi

import data.scutil.token.token_has_role
import data.scutil.rpc.verb_match
import data.scutil.rpc.read_request

default allow := false # take over all permissions for this service

# admin based access is unrestricted
allow {token_has_role("admin")}
allow {token_has_role("super-admin")}
# certificate based access is unrestricted, this may change in future
allow {input.certificate_valid}

# Commissioners can do anything with services
allow {token_has_role("commissioner")}

# Operators are allowed extra privileges to start/stop any service/automation.
# Also operators can fully manage zones as they see fit.
allow {
  token_has_role("operator")
  read_request
}
allow {
  token_has_role("operator")
  verb_match({"Stop", "Start"})
}
allow {
  token_has_role("operator")
  endswith(input.request.name, "/zones")
}
allow {
  token_has_role("operator")
  input.request.name == "zones"
}

allow {
  token_has_role("viewer")
  read_request
}

# signage has no rights here

# Allow anyone to get service metadata about any service.
allow { input.method == "GetServiceMetadata" }
allow { input.method == "PullServiceMetadata" }
