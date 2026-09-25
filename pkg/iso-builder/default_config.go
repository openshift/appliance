package isobuilder

import (
	isobuilderconfig "github.com/openshift/appliance/pkg/iso-builder/config"
)

// Temporary fallback — delete this file once external config injection is available.
func defaultISOBuilderConfig() *isobuilderconfig.Config {
	return &isobuilderconfig.Config{
		OpenshiftVersion: "4.22",
		Architecture:     "x86_64",
		PullSecret:       readPullSecret(),
		AdditionalImages: []string{
			"registry.redhat.io/rhel9/support-tools:latest",
		},
		OLMOperators: []isobuilderconfig.OLMOperator{
			{Name: "kubevirt-hyperconverged", Channel: "stable"},
			{Name: "mtv-operator", Channel: "release-v2.12"},
			{Name: "kubernetes-nmstate-operator", Channel: "stable"},
			{Name: "node-healthcheck-operator", Channel: "stable"},
			{Name: "node-maintenance-operator", Channel: "stable"},
			{Name: "fence-agents-remediation", Channel: "stable"},
			{Name: "cluster-kube-descheduler-operator", Channel: "stable"},
			{Name: "metallb-operator", Channel: "stable"},
			{Name: "cluster-observability-operator", Channel: "stable"},
			{Name: "redhat-oadp-operator", Channel: "stable"},
			{Name: "local-storage-operator", Channel: "stable"},
			{Name: "lvms-operator", Channel: "stable-4.22"},
			{Name: "numaresources-operator", Channel: "4.22"},
			{Name: "loki-operator", Channel: "stable-6.6"},
			{Name: "cluster-logging", Channel: "stable-6.6"},
		},
	}
}
