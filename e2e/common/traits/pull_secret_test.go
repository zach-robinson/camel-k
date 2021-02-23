// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package traits

import (
	"testing"

	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/openshift"
)

func TestPullSecretTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		ocp, err := openshift.IsOpenShift(TestClient())
		Expect(err).To(BeNil())

		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		t.Run("Image pull secret is set on pod", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/Java.java",
				"-t", "pull-secret.enabled=true",
				"-t", "pull-secret.secret-name=dummy-secret").Execute()).To(Succeed())
			// pod may not run because the pull secret is dummy
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Or(Equal(v1.PodRunning), Equal(v1.PodPending)))

			pod := IntegrationPod(ns, "java")()
			Expect(pod.Spec.ImagePullSecrets).NotTo(BeEmpty())
			Expect(pod.Spec.ImagePullSecrets[0].Name).To(Equal("dummy-secret"))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Explicity disable image pull secret", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/Java.java",
				"-t", "pull-secret.enabled=false").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "java", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(ns, "java")()
			if ocp {
				// OpenShift `default` service account has imagePullSecrets so it's always set
				Expect(pod.Spec.ImagePullSecrets).NotTo(BeEmpty())
			} else {
				Expect(pod.Spec.ImagePullSecrets).To(BeNil())
			}

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		if ocp {
			// OpenShift always has an internal registry so image pull secret is set by default
			t.Run("Image pull secret is automatically set by default", func(t *testing.T) {
				Expect(Kamel("run", "-n", ns, "files/Java.java").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(v1.PodRunning))
				Eventually(IntegrationCondition(ns, "java", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
				Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

				pod := IntegrationPod(ns, "java")()
				Expect(pod.Spec.ImagePullSecrets).NotTo(BeEmpty())
				Expect(pod.Spec.ImagePullSecrets[0].Name).To(HavePrefix("default-dockercfg-"))

				Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
			})
		}
	})
}
