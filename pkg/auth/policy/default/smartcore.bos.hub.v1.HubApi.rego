package smartcore.bos.hub.v1.HubApi

import data.scutil.rpc.read_request
import data.scutil.rpc.verb_match

# Allow anybody to request information about nodes.
# This is useful for status monitoring.
allow if read_request

allow if {
  verb_match({"Inspect", "Test"})
  not sends_setup_code
}

# The hub passes a setup code on to the node, and every wrong code uses up one of the node's limited guesses.
# Left open, anybody who can reach the hub could lock the installer out of nodes they can't reach themselves.
# So inspecting with a code falls through to the general rules, which are the same ones that let you enrol a node.
sends_setup_code if object.get(input.request, "setupCode", "") != ""
