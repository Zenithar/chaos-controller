// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package v1beta1_test

import (
	. "github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OutOfPodsSpec", func() {
	var spec *OutOfPodsSpec

	BeforeEach(func() {
		spec = &OutOfPodsSpec{}
	})

	Describe("Validate", func() {
		Context("with default spec", func() {
			It("should return no error", func() {
				Expect(spec.Validate()).To(Succeed())
			})
		})

		Context("with forced enabled", func() {
			BeforeEach(func() {
				spec.Forced = true
			})

			It("should return no error", func() {
				Expect(spec.Validate()).To(Succeed())
			})
		})
	})

	Describe("GenerateArgs", func() {
		Context("with forced disabled", func() {
			BeforeEach(func() {
				spec.Forced = false
			})

			It("should return args without --forced flag", func() {
				args := spec.GenerateArgs()
				Expect(args).To(Equal([]string{"out-of-pods"}))
			})
		})

		Context("with forced enabled", func() {
			BeforeEach(func() {
				spec.Forced = true
			})

			It("should return args with --forced flag", func() {
				args := spec.GenerateArgs()
				Expect(args).To(Equal([]string{"out-of-pods", "--forced"}))
			})
		})
	})

	Describe("Explain", func() {
		Context("with forced disabled", func() {
			BeforeEach(func() {
				spec.Forced = false
			})

			It("should return explanation mentioning NoSchedule", func() {
				explanation := spec.Explain()
				Expect(explanation).To(HaveLen(2))
				Expect(explanation[0]).To(Equal(""))
				Expect(explanation[1]).To(ContainSubstring("NoSchedule"))
				Expect(explanation[1]).To(ContainSubstring("prevents new pods from being scheduled"))
				Expect(explanation[1]).ToNot(ContainSubstring("evicts existing pods"))
			})
		})

		Context("with forced enabled", func() {
			BeforeEach(func() {
				spec.Forced = true
			})

			It("should return explanation mentioning NoExecute and eviction", func() {
				explanation := spec.Explain()
				Expect(explanation).To(HaveLen(2))
				Expect(explanation[0]).To(Equal(""))
				Expect(explanation[1]).To(ContainSubstring("NoExecute"))
				Expect(explanation[1]).To(ContainSubstring("evicts existing pods"))
			})
		})
	})
})
