package smartcore.bos.AlertApi

import data.scutil.token.token_has_role
import data.scutil.rpc.verb_match

allow {
  token_has_role("operator")
  verb_match({"Acknowledge", "Unacknowledge"})
}