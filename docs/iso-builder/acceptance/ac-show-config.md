# ac-show-config acceptance criteria

# scenario 1: Missing config file
Given an iso-builder instance without an embedded configuration
When the user executes the show-config command
Then it fails with error