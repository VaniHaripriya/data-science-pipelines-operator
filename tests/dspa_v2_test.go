//go:build test_integration

/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"fmt"
	"testing"
	"time"

	testUtil "github.com/opendatahub-io/data-science-pipelines-operator/tests/util"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (suite *IntegrationTestSuite) TestDSPADeployment() {
	podCount := 8
	if suite.DSPA.Spec.ObjectStorage.ExternalStorage != nil {
		podCount = podCount - 1
	}
	if suite.DSPA.Spec.Database.ExternalDB != nil {
		podCount = podCount - 1
	}
	deployments := []string{
		fmt.Sprintf("ds-pipeline-%s", suite.DSPA.Name),
		fmt.Sprintf("ds-pipeline-persistenceagent-%s", suite.DSPA.Name),
		fmt.Sprintf("ds-pipeline-scheduledworkflow-%s", suite.DSPA.Name),
	}

	if suite.DSPA.Spec.ObjectStorage.ExternalStorage == nil && suite.DSPA.Spec.Database.ExternalDB == nil {
		deployments = append(deployments,
			fmt.Sprintf("mariadb-%s", suite.DSPA.Name),
			fmt.Sprintf("minio-%s", suite.DSPA.Name),
		)
	}
	suite.T().Run("with default MariaDB and Minio", func(t *testing.T) {
		t.Run(fmt.Sprintf("should have %d pods", podCount), func(t *testing.T) {
			timeout := time.Second * 120
			interval := time.Second * 2
			actualPodCount := 0

			require.Eventually(t, func() bool {
				podList := &corev1.PodList{}
				// retrieve the running pods only, to allow for multiple reruns of  the test suite
				listOpts := []client.ListOption{
					client.InNamespace(suite.DSPANamespace),
					client.MatchingFields{"status.phase": string(corev1.PodRunning)},
				}
				err := suite.Clientmgr.k8sClient.List(suite.Ctx, podList, listOpts...)
				require.NoError(suite.T(), err)
				actualPodCount = len(podList.Items)

				// Print out pod statuses for troubleshooting
				if podCount != actualPodCount {
					t.Log(fmt.Sprintf("expected %d pods to successfully deploy, got %d instead. Pods in the namespace:", podCount, actualPodCount))
					totalPodList := &corev1.PodList{}
					listOpts1 := []client.ListOption{
						client.InNamespace(suite.DSPANamespace),
					}
					err1 := suite.Clientmgr.k8sClient.List(suite.Ctx, totalPodList, listOpts1...)
					require.NoError(t, err1)
					for _, pod := range totalPodList.Items {
						t.Log(fmt.Sprintf("Pod Name: %s, Status: %s", pod.Name, pod.Status.Phase))
					}
					return false
				} else {
					return true
				}
			}, timeout, interval)
		})
		for _, deployment := range deployments {
			t.Run(fmt.Sprintf("should have a ready %s deployment", deployment), func(t *testing.T) {
				testUtil.TestForSuccessfulDeployment(t, suite.Ctx, suite.DSPANamespace, deployment, suite.Clientmgr.k8sClient)
			})
		}
	})
}

func (suite *IntegrationTestSuite) TestDSPADeploymentWithK8sNativeApi() {
	podCount := 8
	if suite.DSPA.Spec.ObjectStorage.ExternalStorage != nil {
		podCount = podCount - 1
	}
	if suite.DSPA.Spec.Database.ExternalDB != nil {
		podCount = podCount - 1
	}
	deployments := []string{
		fmt.Sprintf("ds-pipeline-%s", suite.DSPA.Name),
		fmt.Sprintf("ds-pipeline-persistenceagent-%s", suite.DSPA.Name),
		fmt.Sprintf("ds-pipeline-scheduledworkflow-%s", suite.DSPA.Name),
	}

	if suite.DSPA.Spec.ObjectStorage.ExternalStorage == nil && suite.DSPA.Spec.Database.ExternalDB == nil {
		deployments = append(deployments,
			fmt.Sprintf("mariadb-%s", suite.DSPA.Name),
			fmt.Sprintf("minio-%s", suite.DSPA.Name),
		)
	}

	// Sets the suite DSPA to deploy DSP with the K8s Pipeline Store set.
	suite.DSPA.Spec.APIServer.PipelineStorage = "kubernetes"

	suite.T().Run("with default MariaDB and Minio", func(t *testing.T) {
		t.Run(fmt.Sprintf("should have %d pods", podCount), func(t *testing.T) {
			timeout := time.Second * 120
			interval := time.Second * 2
			actualPodCount := 0

			// First check if all pods are running
			require.Eventually(t, func() bool {
				podList := &corev1.PodList{}
				listOpts := []client.ListOption{
					client.InNamespace(suite.DSPANamespace),
					client.MatchingFields{"status.phase": string(corev1.PodRunning)},
				}
				err := suite.Clientmgr.k8sClient.List(suite.Ctx, podList, listOpts...)
				require.NoError(t, err)
				actualPodCount = len(podList.Items)

				if podCount != actualPodCount {
					t.Log(fmt.Sprintf("expected %d pods to successfully deploy, got %d instead. Pods in the namespace:", podCount, actualPodCount))
					totalPodList := &corev1.PodList{}
					listOpts1 := []client.ListOption{
						client.InNamespace(suite.DSPANamespace),
					}
					err1 := suite.Clientmgr.k8sClient.List(suite.Ctx, totalPodList, listOpts1...)
					require.NoError(t, err1)
					for _, pod := range totalPodList.Items {
						t.Log(fmt.Sprintf("Pod Name: %s, Status: %s", pod.Name, pod.Status.Phase))
					}
					return false
				}
				return true
			}, timeout, interval)

			t.Run("should have API Server configured with K8s Native API", func(t *testing.T) {
				require.Eventually(t, func() bool {
					totalPodList := &corev1.PodList{}
					listOpts1 := []client.ListOption{
						client.InNamespace(suite.DSPANamespace),
						client.MatchingLabels{
							"app": deployments[0],
						},
					}
					err1 := suite.Clientmgr.k8sClient.List(suite.Ctx, totalPodList, listOpts1...)
					require.NoError(t, err1)

					if len(totalPodList.Items) == 0 {
						t.Log("No pods found with app label:", deployments[0])
						return false
					}

					for _, pod := range totalPodList.Items {
						t.Log(fmt.Sprintf("Checking pod: %s", pod.Name))
						if pod.Status.Phase != corev1.PodRunning {
							t.Log(fmt.Sprintf("Pod %s is not running (status: %s)", pod.Name, pod.Status.Phase))
							continue
						}

						for _, arg := range pod.Spec.Containers[0].Args {
							if arg == "--pipelinesStoreKubernetes=true" {
								t.Log(fmt.Sprintf("Found API Server pod %s with K8s Native API configured", pod.Name))
								return true
							}
						}
						t.Log(fmt.Sprintf("Pod %s found but missing --pipelinesStoreKubernetes=true flag", pod.Name))
					}
					return false
				}, timeout, interval)
			})
		})

		for _, deployment := range deployments {
			t.Run(fmt.Sprintf("should have a ready %s deployment", deployment), func(t *testing.T) {
				testUtil.TestForSuccessfulDeployment(t, suite.Ctx, suite.DSPANamespace, deployment, suite.Clientmgr.k8sClient)
			})
		}
	})
}
