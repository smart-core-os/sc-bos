# Applies to every service in the package: AccessCredentialApi and AccessCredentialInfo.
# Credential values, like card numbers, can be used to clone a credential,
# so tenant trait:read and trait:write permissions are not enough.
package smartcore.bos.accesscredential.v1

import data.scutil.rpc.read_request
import data.scutil.rpc.write_request
import data.scutil.token.token_has_role

default allow := false

# Unrestricted access for admin roles and valid certificates.
allow if token_has_role("admin")
allow if token_has_role("super-admin")
allow if input.certificate_valid
allow if token_has_role("commissioner")

# Operators may read and manage credentials.
allow if {
	token_has_role("operator")
	read_request
}
allow if {
	token_has_role("operator")
	write_request
}
