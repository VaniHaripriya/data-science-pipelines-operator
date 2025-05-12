//go:build test_all || test_unit

/*

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

package controllers

import (
	"testing"

	dspav1 "github.com/opendatahub-io/data-science-pipelines-operator/api/v1"
	"github.com/opendatahub-io/data-science-pipelines-operator/controllers/testutil"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func TestDeployWebhook(t *testing.T) {
	dspa := testutil.CreateTestDSPA()

	// Create Context, Fake Controller and Params
	ctx, params, reconciler := CreateNewTestObjects()
	err := params.ExtractParams(ctx, dspa, reconciler.Client, reconciler.Log)
	assert.Nil(t, err)

	params.DSPONamespace = "testDSPONamespace"
	params.WebhookName = "kubernetes-webhook"

	dspoDeployment := testutil.CreateTestDSPODeployment(params.DSPONamespace)
	err = reconciler.Client.Create(ctx, dspoDeployment)
	assert.NoError(t, err)

	// Assert Webhook Deployment doesn't yet exist
	deployment := &appsv1.Deployment{}
	created, err := reconciler.IsResourceCreated(ctx, deployment, "kubernetes-webhook", "testDSPONamespace")
	assert.False(t, created)
	assert.Nil(t, err)

	// Run test reconciliation
	err = reconciler.ReconcileWebhook(ctx, dspa, params)
	assert.Nil(t, err)

	// Assert APIServer Deployment now exists
	deployment = &appsv1.Deployment{}
	created, err = reconciler.IsResourceCreated(ctx, deployment, "kubernetes-webhook", "testDSPONamespace")
	assert.True(t, created)
	assert.Nil(t, err)
}

func TestWebhookNotDeployedWithoutKubernetesPipelineStorage(t *testing.T) {
	dspa := testutil.CreateEmptyDSPA()
	dspa.Name = "dspa-non-k8s"
	dspa.Namespace = "testnamespace"
	dspa.Spec.APIServer = &dspav1.APIServer{
		Deploy:          true,
		PipelineStorage: "minio",
	}

	ctx, params, reconciler := CreateNewTestObjects()
	err := params.ExtractParams(ctx, dspa, reconciler.Client, reconciler.Log)
	assert.Nil(t, err)

	params.DSPONamespace = "testDSPONamespace"
	params.WebhookName = "kubernetes-webhook"

	dspoDeployment := testutil.CreateTestDSPODeployment(params.DSPONamespace)
	err = reconciler.Client.Create(ctx, dspoDeployment)
	assert.NoError(t, err)

	err = reconciler.Client.Create(ctx, dspa)
	assert.NoError(t, err)

	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      dspa.Name,
			Namespace: dspa.Namespace,
		},
	})
	assert.NoError(t, err)

	// Webhook Deployment should NOT be created
	deployment := &appsv1.Deployment{}
	created, err := reconciler.IsResourceCreated(ctx, deployment, "kubernetes-webhook", "testDSPONamespace")
	assert.False(t, created)
	assert.Nil(t, err)
}

func TestWebhookRemovedWhenLastKubernetesDSPADeleted(t *testing.T) {
	dspa := testutil.CreateTestDSPA()

	ctx, params, reconciler := CreateNewTestObjects()
	err := params.ExtractParams(ctx, dspa, reconciler.Client, reconciler.Log)
	assert.Nil(t, err)

	params.DSPONamespace = "testDSPONamespace"
	params.WebhookName = "kubernetes-webhook"

	dspoDeployment := testutil.CreateTestDSPODeployment(params.DSPONamespace)
	err = reconciler.Client.Create(ctx, dspoDeployment)
	assert.NoError(t, err)

	controllerutil.AddFinalizer(dspa, finalizerName)

	// Create the DSPA resource in the fake client
	err = reconciler.Client.Create(ctx, dspa)
	assert.NoError(t, err)

	// First reconcile creates the webhook
	err = reconciler.ReconcileWebhook(ctx, dspa, params)
	assert.Nil(t, err)

	deployment := &appsv1.Deployment{}
	created, err := reconciler.IsResourceCreated(ctx, deployment, "kubernetes-webhook", "testDSPONamespace")
	assert.True(t, created)
	assert.Nil(t, err)

	// Simulate deletion of last DSPA with kubernetes storage
	err = reconciler.Client.Delete(ctx, dspa)
	assert.NoError(t, err)

	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      dspa.Name,
			Namespace: dspa.Namespace,
		},
	})
	assert.NoError(t, err)

	// Webhook should now be deleted
	deployment = &appsv1.Deployment{}
	created, err = reconciler.IsResourceCreated(ctx, deployment, "kubernetes-webhook", "testDSPONamespace")
	assert.False(t, created)
	assert.Nil(t, err)
}

func TestWebhookPersistsWhenNonKubernetesDSPADeleted(t *testing.T) {
	// First: create a DSPA with kubernetes PipelineStorage to ensure webhook exists
	k8sDSPA := testutil.CreateTestDSPA()
	ctx, params, reconciler := CreateNewTestObjects()
	err := params.ExtractParams(ctx, k8sDSPA, reconciler.Client, reconciler.Log)
	assert.Nil(t, err)

	params.DSPONamespace = "testDSPONamespace"
	params.WebhookName = "kubernetes-webhook"

	dspoDeployment := testutil.CreateTestDSPODeployment(params.DSPONamespace)
	err = reconciler.Client.Create(ctx, dspoDeployment)
	assert.NoError(t, err)

	controllerutil.AddFinalizer(k8sDSPA, finalizerName)

	// Create the DSPA resource in the fake client
	err = reconciler.Client.Create(ctx, k8sDSPA)
	assert.NoError(t, err)

	err = reconciler.ReconcileWebhook(ctx, k8sDSPA, params)
	assert.Nil(t, err)

	deployment := &appsv1.Deployment{}
	created, err := reconciler.IsResourceCreated(ctx, deployment, "kubernetes-webhook", "testDSPONamespace")
	assert.True(t, created)
	assert.Nil(t, err)

	//Create a DSPA with non-kubernetes PipelineStorage
	nonK8sDSPA := testutil.CreateEmptyDSPA()
	nonK8sDSPA.Name = "dspa-non-k8s"
	nonK8sDSPA.Namespace = "testnamespace"
	nonK8sDSPA.Spec.APIServer = &dspav1.APIServer{
		Deploy:          true,
		PipelineStorage: "minio",
	}

	err = reconciler.Client.Create(ctx, nonK8sDSPA)
	assert.NoError(t, err)

	// Simulate deletion of DSPA with non-kubernetes storage
	err = reconciler.Client.Delete(ctx, nonK8sDSPA)
	assert.NoError(t, err)

	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      nonK8sDSPA.Name,
			Namespace: nonK8sDSPA.Namespace,
		},
	})
	assert.NoError(t, err)

	// Webhook should still exist
	deployment = &appsv1.Deployment{}
	created, err = reconciler.IsResourceCreated(ctx, deployment, "kubernetes-webhook", "testDSPONamespace")
	assert.True(t, created)
	assert.Nil(t, err)
}
