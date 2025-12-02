// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package injector

import (
	"context"
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/o11y/tags"
	"github.com/DataDog/chaos-controller/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// OutOfPodsTaintKey is the taint key used to simulate out of pods condition
	OutOfPodsTaintKey = types.GroupName + "/out-of-pods"
)

// outOfPodsInjector describes an out of pods injector
type outOfPodsInjector struct {
	spec   v1beta1.OutOfPodsSpec
	config OutOfPodsInjectorConfig
}

// OutOfPodsInjectorConfig contains needed drivers to
// create an OutOfPodsInjector
type OutOfPodsInjectorConfig struct {
	Config
}

// NewOutOfPodsInjector creates an OutOfPodsInjector object with the given config
func NewOutOfPodsInjector(spec v1beta1.OutOfPodsSpec, config OutOfPodsInjectorConfig) Injector {
	return &outOfPodsInjector{
		spec:   spec,
		config: config,
	}
}

func (i *outOfPodsInjector) TargetName() string {
	return i.config.TargetName()
}

func (i *outOfPodsInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindOutOfPods
}

// Inject applies a taint to the target node to simulate out of pods condition
func (i *outOfPodsInjector) Inject() error {
	nodeName := i.config.Disruption.TargetNodeName

	// Determine the taint effect based on the spec
	var effect corev1.TaintEffect
	if i.spec.Forced {
		effect = corev1.TaintEffectNoExecute
	} else {
		effect = corev1.TaintEffectNoSchedule
	}

	i.config.Log.Infow("injecting out of pods disruption by applying node taint",
		tags.NodeNameKey, nodeName,
		tags.TaintEffectKey, effect,
	)

	if i.config.Disruption.DryRun {
		i.config.Log.Infow("dry-run mode: would apply taint to node",
			tags.NodeNameKey, nodeName,
			tags.TaintKeyKey, OutOfPodsTaintKey,
			tags.TaintEffectKey, effect,
		)
		return nil
	}

	// Get the node
	node, err := i.config.K8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting node %s: %w", nodeName, err)
	}

	// Check if taint already exists
	for _, taint := range node.Spec.Taints {
		if taint.Key == OutOfPodsTaintKey {
			i.config.Log.Infow("taint already exists on node, skipping",
				tags.NodeNameKey, nodeName,
				tags.TaintKeyKey, OutOfPodsTaintKey,
			)
			return nil
		}
	}

	// Add the taint
	newTaint := corev1.Taint{
		Key:    OutOfPodsTaintKey,
		Value:  "true",
		Effect: effect,
	}
	node.Spec.Taints = append(node.Spec.Taints, newTaint)

	// Update the node
	_, err = i.config.K8sClient.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating node %s with taint: %w", nodeName, err)
	}

	i.config.Log.Infow("successfully applied out of pods taint to node",
		tags.NodeNameKey, nodeName,
		tags.TaintKeyKey, OutOfPodsTaintKey,
		tags.TaintEffectKey, effect,
	)

	return nil
}

func (i *outOfPodsInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

// Clean removes the out of pods taint from the target node
func (i *outOfPodsInjector) Clean() error {
	nodeName := i.config.Disruption.TargetNodeName

	i.config.Log.Infow("cleaning out of pods disruption by removing node taint",
		tags.NodeNameKey, nodeName,
	)

	if i.config.Disruption.DryRun {
		i.config.Log.Infow("dry-run mode: would remove taint from node",
			tags.NodeNameKey, nodeName,
			tags.TaintKeyKey, OutOfPodsTaintKey,
		)
		return nil
	}

	// Get the node
	node, err := i.config.K8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting node %s: %w", nodeName, err)
	}

	// Find and remove the taint
	newTaints := []corev1.Taint{}
	taintFound := false

	for _, taint := range node.Spec.Taints {
		if taint.Key == OutOfPodsTaintKey {
			taintFound = true
			continue
		}
		newTaints = append(newTaints, taint)
	}

	if !taintFound {
		i.config.Log.Infow("taint not found on node, nothing to clean",
			tags.NodeNameKey, nodeName,
			tags.TaintKeyKey, OutOfPodsTaintKey,
		)
		return nil
	}

	node.Spec.Taints = newTaints

	// Update the node
	_, err = i.config.K8sClient.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating node %s to remove taint: %w", nodeName, err)
	}

	i.config.Log.Infow("successfully removed out of pods taint from node",
		tags.NodeNameKey, nodeName,
		tags.TaintKeyKey, OutOfPodsTaintKey,
	)

	return nil
}
