package smartcore.bos.setup.v1.SetupApi

default allow := false

# GetSetupState is open so a client can tell whether to show the setup wizard before anyone can log in.
allow if input.method == "GetSetupState"

# CompleteSetup carries the setup code in its request and checks it, with a limit on wrong guesses.
# The node refuses it once claimed.
allow if input.method == "CompleteSetup"
