package smartcore.bos.HubApi

import data.scutil.rpc.read_request
import data.scutil.rpc.verb_match

# Allow anybody to request information about nodes.
# This is useful for status monitoring.
allow { read_request }

allow {
  verb_match({"Inspect", "Test"})
}