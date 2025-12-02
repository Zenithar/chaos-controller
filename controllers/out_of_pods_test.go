// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package controllers

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

var _ = Describe("OutOfPods Disruption", func() {
	var (
		disruption chaosv1beta1.Disruption
	)

	BeforeEach(func(ctx SpecContext) {
		disruption = chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    namespace,
				GenerateName: "out-of-pods-test-",
				Annotations:  map[string]string{chaosv1beta1.SafemodeEnvironmentAnnotation: "lima"},
			},
			Spec: chaosv1beta1.DisruptionSpec{
				DryRun: false,
				Count:  &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				Unsafemode: &chaosv1beta1.UnsafemodeSpec{
					DisableAll: true,
				},
				Selector: map[string]string{"kubernetes.io/hostname": clusterName},
				Level:    chaostypes.DisruptionLevelNode,
				Duration: shortDisruptionDuration,
				OutOfPods: &chaosv1beta1.OutOfPodsSpec{
					Forced: false,
				},
			},
		}
	})

	// Helper function to verify node has the expected taint
	verifyNodeTaint := func(ctx SpecContext, nodeName string, expectedEffect corev1.TaintEffect, shouldExist bool) {
		GinkgoHelper()

		Eventually(func(ctx SpecContext) error {
			node := &corev1.Node{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: nodeName}, node); err != nil {
				return fmt.Errorf("failed to get node %s: %w", nodeName, err)
			}

			taintFound := false
			var foundEffect corev1.TaintEffect

			for _, taint := range node.Spec.Taints {
				if taint.Key == injector.OutOfPodsTaintKey {
					taintFound = true
					foundEffect = taint.Effect
					break
				}
			}

			if shouldExist {
				if !taintFound {
					return fmt.Errorf("expected taint %s not found on node %s", injector.OutOfPodsTaintKey, nodeName)
				}
				if foundEffect != expectedEffect {
					return fmt.Errorf("expected taint effect %s but got %s", expectedEffect, foundEffect)
				}
			} else {
				if taintFound {
					return fmt.Errorf("taint %s should not exist on node %s but was found with effect %s", injector.OutOfPodsTaintKey, nodeName, foundEffect)
				}
			}

			return nil
		}).WithContext(ctx).Within(calcDisruptionGoneTimeout(disruption)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())
	}

	Context("Basic OutOfPods with NoSchedule taint", func() {
		It("should apply and remove NoSchedule taint on the node", func(ctx SpecContext) {
			By("Creating a dummy pod to satisfy the CreateDisruption helper")
			targetPod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "out-of-pods-dummy-",
					Namespace:    namespace,
					Labels:       map[string]string{"app": "out-of-pods-test"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "k8s.gcr.io/pause:3.4.1",
							Name:  "pause",
						},
					},
				},
			}
			dummyPod := <-CreateRunningPod(ctx, targetPod)

			By("Creating the OutOfPods disruption")
			disruption = CreateDisruption(ctx, disruption, dummyPod)

			By("Waiting for disruption to be injected")
			ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusInjected)

			By("Verifying chaos pod is created")
			ExpectChaosPods(ctx, disruption, 1)

			By("Verifying node has NoSchedule taint")
			verifyNodeTaint(ctx, clusterName, corev1.TaintEffectNoSchedule, true)

			By("Deleting the disruption")
			DeleteDisruption(ctx, disruption)

			By("Verifying taint is removed after cleanup")
			verifyNodeTaint(ctx, clusterName, corev1.TaintEffectNoSchedule, false)
		})
	})

	Context("Forced OutOfPods with NoExecute taint", func() {
		BeforeEach(func() {
			disruption.Spec.OutOfPods.Forced = true
		})

		It("should apply and remove NoExecute taint on the node", func(ctx SpecContext) {
			By("Creating a dummy pod to satisfy the CreateDisruption helper")
			targetPod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "out-of-pods-forced-dummy-",
					Namespace:    namespace,
					Labels:       map[string]string{"app": "out-of-pods-forced-test"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "k8s.gcr.io/pause:3.4.1",
							Name:  "pause",
						},
					},
				},
			}
			dummyPod := <-CreateRunningPod(ctx, targetPod)

			By("Creating the OutOfPods disruption with forced=true")
			disruption = CreateDisruption(ctx, disruption, dummyPod)

			By("Waiting for disruption to be injected")
			ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusInjected)

			By("Verifying chaos pod is created")
			ExpectChaosPods(ctx, disruption, 1)

			By("Verifying node has NoExecute taint")
			verifyNodeTaint(ctx, clusterName, corev1.TaintEffectNoExecute, true)

			By("Deleting the disruption")
			DeleteDisruption(ctx, disruption)

			By("Verifying taint is removed after cleanup")
			verifyNodeTaint(ctx, clusterName, corev1.TaintEffectNoExecute, false)
		})
	})

	Context("Dry run mode", func() {
		BeforeEach(func() {
			disruption.Spec.DryRun = true
		})

		It("should not apply taint in dry run mode", func(ctx SpecContext) {
			By("Creating a dummy pod to satisfy the CreateDisruption helper")
			targetPod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "out-of-pods-dryrun-dummy-",
					Namespace:    namespace,
					Labels:       map[string]string{"app": "out-of-pods-dryrun-test"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "k8s.gcr.io/pause:3.4.1",
							Name:  "pause",
						},
					},
				},
			}
			dummyPod := <-CreateRunningPod(ctx, targetPod)

			By("Creating the OutOfPods disruption in dry run mode")
			disruption = CreateDisruption(ctx, disruption, dummyPod)

			By("Waiting for disruption to be injected")
			ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusInjected)

			By("Verifying chaos pod is created")
			ExpectChaosPods(ctx, disruption, 1)

			By("Verifying node does NOT have the taint (dry run mode)")
			verifyNodeTaint(ctx, clusterName, corev1.TaintEffectNoSchedule, false)

			By("Deleting the disruption")
			DeleteDisruption(ctx, disruption)
		})
	})
})
