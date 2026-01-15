package smartcore.bos.EnrollmentApi

import data.scutil.token.token_has_role
import data.scutil.rpc.read_request
import data.scutil.rpc.verb_match

# Allow anybody to request information about their enrollment.
# This is useful for status monitoring.
allow { read_request }
allow { input.method == "TestEnrollment" }
