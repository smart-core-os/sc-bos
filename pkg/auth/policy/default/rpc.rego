package scutil.rpc

read_verbs := {
  "Get",
  "List",
  "Pull",
  "Describe"
}

write_verbs := {
  "Create",
  "Update",
  "Delete"
}

verb_match(verbs) if regex.match(concat("", ["^(", concat("|", verbs), ")[A-Z]"]), input.method)

read_request if verb_match(read_verbs)

write_request if verb_match(write_verbs)

rpc_match(service, method) if {
  input.service == service
  input.method == method
}

rpc_match_methods(service, methods) if {
  input.service == service
  input.method in methods
}

rpc_match_verbs(service, verbs) if {
  input.service == service
  verb_match(verbs)
}
