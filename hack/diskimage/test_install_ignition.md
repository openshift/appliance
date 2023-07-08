# Test changes in InstallIgnition asset

## Preparation steps:    
1. Build appliance with --debug-bootstrap flag
2. Boot appliance and wait for "Ironic will reboot the node shortly" in assisted-service logs.
   - Leaves the appliance in pre-installed state (bootstrap completed)
3. Shutdown appliance
4. Run hack/diskimage/extract_install_ignition.sh
   - Extracts base ignition config to assets/ignition/base/config.ign
5. Run hack/diskimage/convert_to_qcow2.sh
    
## To test changes:
1. Run 'generate-install-ignition' command
   - Generates merged ignition (base ignition from step 4 + InstallIgnition asset)
   - Outputs to assets/ignition/install/config.ign
2. Run hack/diskimage/embed_install_ignition.sh
   - Creates a snapshot and embeds the merged ignition
3. Run the appliance
