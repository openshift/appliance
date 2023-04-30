mapfile -t release_images < <( oc adm release info quay.io/openshift-release-dev/ocp-release@sha256:${OPENSHIFT_RELEASE_TAG} -o json | jq -r '.references.spec.tags[] | .name + " " + .from.name' )
echo "Getting release_images for digest"

   cat > "${imageset}" << EOF
kind: ImageSetConfiguration
apiVersion: mirror.openshift.io/v1alpha2
archiveSize: 8
storageConfig:
  local:
    path: metadata
mirror:
  platform:
    architectures:
      - "amd64"
    channels:
      - name: candidate-${OPENSHIFT_RELEASE_STREAM}
        minVersion: $latest_release
        maxVersion: $latest_release
        type: ocp
  additionalImages:
    - name: registry.redhat.io/ubi8/ubi:latest
  blockedImages:
EOF

   for image_info in "${release_images[@]}"; do
        read -r image_name image_ref <<< $image_info
        case "$image_name" in agent-installer-api-server | must-gather | hyperkube | cloud-credential-operator | cluster-policy-controller | agent-installer-orchestrator | pod | cluster-config-operator | cluster-etcd-operator | cluster-kube-controller-manager-operator | cluster-kube-scheduler-operator | agent-installer-node-agent | machine-config-operator | etcd | cluster-bootstrap | cluster-ingress-operator | cluster-kube-apiserver-operator | baremetal-installer | keepalived-ipfailover | baremetal-runtimecfg | coredns | installer)
                        >&2 echo "Not blocking $image_name";;
                *)
                        >&2 echo "Blocking $image_name"
                        cat >> "${imageset}" <<EOF
    - name: "$image_ref"
EOF
                        ;;
        esac
   done