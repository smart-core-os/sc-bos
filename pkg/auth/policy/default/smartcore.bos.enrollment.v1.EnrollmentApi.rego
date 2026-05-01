package smartcore.bos.enrollment.v1.EnrollmentApi

import data.scutil.rpc.read_request

# Allow anybody to request information about their enrollment.
# This is useful for status monitoring.
allow if read_request
allow if input.method == "TestEnrollment"
