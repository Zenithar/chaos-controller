// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package injector_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/DataDog/chaos-controller/injector"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

var _ = Describe("OutOfPods", func() {
	var (
		config     OutOfPodsInjectorConfig
		k8sClient  kubernetes.Interface
		inj        Injector
		spec       v1beta1.OutOfPodsSpec
		targetNode *corev1.Node
		nodeName   = "test-node"
	)

	BeforeEach(func() {
		// Create test node
		targetNode = &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeName,
			},
			Spec: corev1.NodeSpec{
				Taints: []corev1.Taint{},
			},
		}

		// Create fake Kubernetes client with test resources
		k8sClient = fake.NewSimpleClientset(targetNode)

		config = OutOfPodsInjectorConfig{
			Config: Config{
				Log:         log,
				MetricsSink: ms,
				K8sClient:   k8sClient,
				Disruption: chaosapi.DisruptionArgs{
					TargetNodeName: nodeName,
					Level:          chaostypes.DisruptionLevelNode,
					DryRun:         false,
				},
			},
		}

		spec = v1beta1.OutOfPodsSpec{
			Forced: false,
		}
	})

	JustBeforeEach(func() {
		inj = NewOutOfPodsInjector(spec, config)
		Expect(inj).ToNot(BeNil())
	})

	Describe("GetDisruptionKind", func() {
		It("should return the correct disruption kind", func() {
			Expect(string(inj.GetDisruptionKind())).To(Equal(chaostypes.DisruptionKindOutOfPods))
		})
	})

	Describe("TargetName", func() {
		It("should return the target node name", func() {
			Expect(inj.TargetName()).To(Equal(nodeName))
		})
	})

	Describe("injection", func() {
		Context("with forced disabled (NoSchedule)", func() {
			BeforeEach(func() {
				spec.Forced = false
			})

			JustBeforeEach(func() {
				Expect(inj.Inject()).To(Succeed())
			})

			It("should apply NoSchedule taint to the node", func() {
				node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(node.Spec.Taints).To(HaveLen(1))
				Expect(node.Spec.Taints[0].Key).To(Equal(OutOfPodsTaintKey))
				Expect(node.Spec.Taints[0].Value).To(Equal("true"))
				Expect(node.Spec.Taints[0].Effect).To(Equal(corev1.TaintEffectNoSchedule))
			})
		})

		Context("with forced enabled (NoExecute)", func() {
			BeforeEach(func() {
				spec.Forced = true
			})

			JustBeforeEach(func() {
				Expect(inj.Inject()).To(Succeed())
			})

			It("should apply NoExecute taint to the node", func() {
				node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(node.Spec.Taints).To(HaveLen(1))
				Expect(node.Spec.Taints[0].Key).To(Equal(OutOfPodsTaintKey))
				Expect(node.Spec.Taints[0].Value).To(Equal("true"))
				Expect(node.Spec.Taints[0].Effect).To(Equal(corev1.TaintEffectNoExecute))
			})
		})

		Context("when taint already exists", func() {
			BeforeEach(func() {
				// Pre-add the taint to the node
				targetNode.Spec.Taints = []corev1.Taint{
					{
						Key:    OutOfPodsTaintKey,
						Value:  "true",
						Effect: corev1.TaintEffectNoSchedule,
					},
				}
				k8sClient = fake.NewSimpleClientset(targetNode)
				config.K8sClient = k8sClient
			})

			JustBeforeEach(func() {
				Expect(inj.Inject()).To(Succeed())
			})

			It("should not add duplicate taint", func() {
				node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(node.Spec.Taints).To(HaveLen(1))
			})
		})

		Context("when node has other taints", func() {
			BeforeEach(func() {
				targetNode.Spec.Taints = []corev1.Taint{
					{
						Key:    "some-other-taint",
						Value:  "value",
						Effect: corev1.TaintEffectNoSchedule,
					},
				}
				k8sClient = fake.NewSimpleClientset(targetNode)
				config.K8sClient = k8sClient
			})

			JustBeforeEach(func() {
				Expect(inj.Inject()).To(Succeed())
			})

			It("should preserve existing taints and add new one", func() {
				node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(node.Spec.Taints).To(HaveLen(2))

				var foundOtherTaint, foundOutOfPodsTaint bool
				for _, taint := range node.Spec.Taints {
					if taint.Key == "some-other-taint" {
						foundOtherTaint = true
					}
					if taint.Key == OutOfPodsTaintKey {
						foundOutOfPodsTaint = true
					}
				}
				Expect(foundOtherTaint).To(BeTrue())
				Expect(foundOutOfPodsTaint).To(BeTrue())
			})
		})
	})

	Describe("dry run mode", func() {
		BeforeEach(func() {
			config.Disruption.DryRun = true
		})

		JustBeforeEach(func() {
			Expect(inj.Inject()).To(Succeed())
		})

		It("should not apply taint to the node", func() {
			node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(node.Spec.Taints).To(BeEmpty())
		})
	})

	Describe("cleanup", func() {
		Context("when taint exists", func() {
			BeforeEach(func() {
				targetNode.Spec.Taints = []corev1.Taint{
					{
						Key:    OutOfPodsTaintKey,
						Value:  "true",
						Effect: corev1.TaintEffectNoSchedule,
					},
				}
				k8sClient = fake.NewSimpleClientset(targetNode)
				config.K8sClient = k8sClient
			})

			JustBeforeEach(func() {
				Expect(inj.Clean()).To(Succeed())
			})

			It("should remove the taint from the node", func() {
				node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(node.Spec.Taints).To(BeEmpty())
			})
		})

		Context("when taint does not exist", func() {
			JustBeforeEach(func() {
				Expect(inj.Clean()).To(Succeed())
			})

			It("should not error and leave node unchanged", func() {
				node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(node.Spec.Taints).To(BeEmpty())
			})
		})

		Context("when node has multiple taints", func() {
			BeforeEach(func() {
				targetNode.Spec.Taints = []corev1.Taint{
					{
						Key:    "some-other-taint",
						Value:  "value",
						Effect: corev1.TaintEffectNoSchedule,
					},
					{
						Key:    OutOfPodsTaintKey,
						Value:  "true",
						Effect: corev1.TaintEffectNoSchedule,
					},
					{
						Key:    "another-taint",
						Value:  "other",
						Effect: corev1.TaintEffectNoExecute,
					},
				}
				k8sClient = fake.NewSimpleClientset(targetNode)
				config.K8sClient = k8sClient
			})

			JustBeforeEach(func() {
				Expect(inj.Clean()).To(Succeed())
			})

			It("should only remove the out-of-pods taint", func() {
				node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(node.Spec.Taints).To(HaveLen(2))

				for _, taint := range node.Spec.Taints {
					Expect(taint.Key).ToNot(Equal(OutOfPodsTaintKey))
				}
			})
		})

		Context("in dry run mode", func() {
			BeforeEach(func() {
				config.Disruption.DryRun = true
				targetNode.Spec.Taints = []corev1.Taint{
					{
						Key:    OutOfPodsTaintKey,
						Value:  "true",
						Effect: corev1.TaintEffectNoSchedule,
					},
				}
				k8sClient = fake.NewSimpleClientset(targetNode)
				config.K8sClient = k8sClient
			})

			JustBeforeEach(func() {
				Expect(inj.Clean()).To(Succeed())
			})

			It("should not remove the taint", func() {
				node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(node.Spec.Taints).To(HaveLen(1))
				Expect(node.Spec.Taints[0].Key).To(Equal(OutOfPodsTaintKey))
			})
		})
	})

	Describe("error handling", func() {
		Context("when node does not exist during injection", func() {
			BeforeEach(func() {
				// Delete the node to simulate missing node
				err := k8sClient.CoreV1().Nodes().Delete(context.Background(), nodeName, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return an error during injection", func() {
				err := inj.Inject()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error getting node"))
			})
		})

		Context("when node does not exist during cleanup", func() {
			BeforeEach(func() {
				// Delete the node to simulate missing node
				err := k8sClient.CoreV1().Nodes().Delete(context.Background(), nodeName, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return an error during cleanup", func() {
				err := inj.Clean()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error getting node"))
			})
		})
	})

	Describe("full injection and cleanup cycle", func() {
		JustBeforeEach(func() {
			Expect(inj.Inject()).To(Succeed())
		})

		It("should add and then remove the taint", func() {
			// Verify taint was added
			node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(node.Spec.Taints).To(HaveLen(1))
			Expect(node.Spec.Taints[0].Key).To(Equal(OutOfPodsTaintKey))

			// Clean up
			Expect(inj.Clean()).To(Succeed())

			// Verify taint was removed
			node, err = k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(node.Spec.Taints).To(BeEmpty())
		})
	})
})
