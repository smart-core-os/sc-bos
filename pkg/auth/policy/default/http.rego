package http

import data.scutil.token.token_has_role

allow if {
  log_level_permission
  input.path == "/__/log/level"
}

allow if {
  # everyone can get the service version
  input.path == "/__/version"
}

allow if {
  pprof_permission
  startswith(input.path, "/__/debug/pprof/")
}

log_level_permission if token_has_role("admin")
log_level_permission if token_has_role("super-admin")

pprof_permission if token_has_role("admin")
pprof_permission if token_has_role("super-admin")
