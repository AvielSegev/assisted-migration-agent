package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	dbContainerName      = "test-planner-db"
	agentContainerName   = "test-planner-agent"
	backendContainerName = "test-planner"
)

var _ = Describe("Agent", Ordered, func() {
	var runner *PodmanRunner

	BeforeAll(func() {
		var err error
		runner, err = NewPodmanRunner(cfg.PodmanSocket)
		Expect(err).To(BeNil())

		_, err = runner.StartContainer(ContainerConfig{
			Name:  dbContainerName,
			Image: "docker.io/library/postgres:17",
			Ports: map[int]int{5432: 5432},
			EnvVars: map[string]string{
				"POSTGRES_USER":     "planner",
				"POSTGRES_PASSWORD": "adminpass",
				"POSTGRES_DB":       "planner",
			},
		})
		Expect(err).To(BeNil())
	})

	AfterAll(func() {
		err := runner.StopContainer(dbContainerName)
		Expect(err).To(BeNil())

		err = runner.RemoveContainer(dbContainerName)
		Expect(err).To(BeNil())
	})

	It("should start", func() {
		Expect(cfg.AgentImage).NotTo(BeEmpty())
	})
})
