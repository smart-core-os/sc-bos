package smartcore.bos.driver.dali.v1.DaliApi

import data.scutil.token.token_has_role

allow if {
  token_has_role("operator")
  input.method in ["StartTest", "StopTest"]
}
