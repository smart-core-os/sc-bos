package smartcore.bos.metadata.v1.MetadataApi

import data.scutil.token.token_has_role

# A hub inspects a node before enrolling it by reading the node's own metadata, which it requests with no name.
# The setup code allows that read and nothing else: not the metadata of any device on the node.
# Every other request falls through to the general rules.
allow if {
  input.method == "GetMetadata"
  token_has_role("setup")
  object.get(input.request, "name", "") == ""
}
