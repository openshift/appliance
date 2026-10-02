# ac-show-config acceptance criteria

# scenario 1: Missing config file
Given an iso-builder instance without an embedded configuration
When the user executes the show-config command
Then it fails with error

# scenario 2: Valid config file
Given an iso-builder instance with a valid embedded configuration
When user executes the show-config command
Then a human-readable and redacted message is displayed for the current configuration 