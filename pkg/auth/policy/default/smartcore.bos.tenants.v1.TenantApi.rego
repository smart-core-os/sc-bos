package smartcore.bos.tenant.TenantApi

import data.scutil.token.token_has_role
import data.scutil.rpc.verb_match

allow {
  token_has_role("operator")
  verb_match({"Add", "Remove", "Regenerate"})
}