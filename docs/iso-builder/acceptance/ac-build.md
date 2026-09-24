# ac-build acceptance criteria

# scenario 1: Missing config file
Given an iso-builder instance without an embedded configuration
When the user executes the build command
Then it fails with error