// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package v1beta1

// OutOfPodsSpec represents an out of pods disruption that simulates node capacity exhaustion
// by applying a taint to the target node
type OutOfPodsSpec struct {
	// Forced when true uses NoExecute taint effect (evicts existing pods),
	// when false uses NoSchedule (prevents new pod scheduling only, default)
	Forced bool `json:"forced,omitempty"`
}

// Validate validates args for the given disruption
func (s *OutOfPodsSpec) Validate() error {
	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *OutOfPodsSpec) GenerateArgs() []string {
	args := []string{
		"out-of-pods",
	}

	if s.Forced {
		args = append(args, "--forced")
	}

	return args
}

// Explain returns a human-readable explanation of what this disruption does
func (s *OutOfPodsSpec) Explain() []string {
	var explanation string
	if s.Forced {
		explanation = "spec.outOfPods.forced applies a NoExecute taint to the target node, " +
			"which prevents new pods from being scheduled AND evicts existing pods that don't tolerate the taint. " +
			"This simulates a node that has completely exhausted its pod capacity."
	} else {
		explanation = "spec.outOfPods applies a NoSchedule taint to the target node, " +
			"which prevents new pods from being scheduled on the node but does not evict existing pods. " +
			"This simulates a node that has reached its pod allocation limit. " +
			"If you'd prefer to also evict existing pods, set outOfPods.forced."
	}

	return []string{"", explanation}
}

