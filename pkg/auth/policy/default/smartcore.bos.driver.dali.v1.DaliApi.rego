package smartcore.bos.driver.dali.DaliApi

import future.keywords.in
import data.scutil.token.token_has_role

allow {
  token_has_role("operator")
  input.method in ["StartTest", "StopTest"]
}