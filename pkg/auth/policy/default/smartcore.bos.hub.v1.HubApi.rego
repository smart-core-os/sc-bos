package smartcore.bos.hub.v1.HubApi

import data.scutil.rpc.read_request
import data.scutil.rpc.verb_match

# Allow anybody to request information about nodes.
# This is useful for status monitoring.
allow if read_request

allow if verb_match({"Inspect", "Test"})
