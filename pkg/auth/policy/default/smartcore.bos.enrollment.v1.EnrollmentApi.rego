package smartcore.bos.enrollment.v1.EnrollmentApi

import data.scutil.rpc.read_request
import data.scutil.token.token_has_role

# Allow anybody to request information about their enrollment.
# This is useful for status monitoring.
allow if read_request
allow if input.method == "TestEnrollment"

# The node's setup code lets a hub enrol it before anyone has an account on it.
# The node refuses a second enrollment, and stops accepting the code once enrolled.
allow if {
  input.method == "CreateEnrollment"
  token_has_role("setup")
}
