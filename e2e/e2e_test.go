package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cat-gate e2e test", func() {
	It("should prepare", func() {
		_, err := kubectl(nil, "apply", "-f", fmt.Sprintf("./manifests/deployment.yaml"))
		Expect(err).Should(Succeed())
	})

	It("should make all pods ready", func() {
		_, err := kubectl(nil, "rollout", "status", "deployment/ubuntu-deployment")
		Expect(err).Should(Succeed())
	})
})
