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
	"strings"
	"testing"

	dspav1 "github.com/opendatahub-io/data-science-pipelines-operator/api/v1"
	"github.com/opendatahub-io/data-science-pipelines-operator/controllers/testutil"
	"github.com/stretchr/testify/assert"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func newWorkflowAdmissionTestDSPA(namespace, name string, deployWorkflowController bool) *dspav1.DataSciencePipelinesApplication {
	return &dspav1.DataSciencePipelinesApplication{
		Spec: dspav1.DSPASpec{
			PodToPodTLS: testutil.BoolPtr(false),
			APIServer:   &dspav1.APIServer{},
			WorkflowController: &dspav1.WorkflowController{
				Deploy: deployWorkflowController,
			},
			Database: &dspav1.Database{
				DisableHealthCheck: false,
				MariaDB: &dspav1.MariaDB{
					Deploy: true,
				},
			},
			MLMD: &dspav1.MLMD{Deploy: true},
			ObjectStorage: &dspav1.ObjectStorage{
				DisableHealthCheck: false,
				Minio: &dspav1.Minio{
					Deploy: false,
					Image:  "someimage",
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func TestReconcileWorkflowAdmissionCreatesPolicyAndBinding(t *testing.T) {
	testNamespace := "testnamespace"
	testDSPAName := "testdspa"

	dspa := newWorkflowAdmissionTestDSPA(testNamespace, testDSPAName, true)
	ctx, params, reconciler := CreateNewTestObjects()

	err := params.ExtractParams(ctx, dspa, reconciler.Client, reconciler.Log)
	assert.Nil(t, err)

	err = reconciler.ReconcileWorkflowAdmission(dspa, params)
	assert.Nil(t, err)

	policy := &admissionregistrationv1.ValidatingAdmissionPolicy{}
	err = reconciler.Get(ctx, types.NamespacedName{Name: "ds-pipeline-workflow-policy-" + testNamespace + "-" + testDSPAName}, policy)
	assert.Nil(t, err)
	assert.Len(t, policy.Spec.Validations, 8)
	assert.Equal(t, "serviceAccountName must be set to "+params.PipelineRunnerServiceAccountName, policy.Spec.Validations[2].Message)

	binding := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
	err = reconciler.Get(ctx, types.NamespacedName{Name: "ds-pipeline-workflow-policy-binding-" + testNamespace + "-" + testDSPAName}, binding)
	assert.Nil(t, err)
	assert.Equal(t, "ds-pipeline-workflow-policy-"+testNamespace+"-"+testDSPAName, binding.Spec.PolicyName)
	assert.Equal(t, []admissionregistrationv1.ValidationAction{admissionregistrationv1.Deny}, binding.Spec.ValidationActions)
}

func TestReconcileWorkflowControllerCreatesPolicyAndNoPodsExecPermission(t *testing.T) {
	testNamespace := "testnamespace"
	testDSPAName := "testdspa"

	dspa := newWorkflowAdmissionTestDSPA(testNamespace, testDSPAName, true)
	ctx, params, reconciler := CreateNewTestObjects()

	err := params.ExtractParams(ctx, dspa, reconciler.Client, reconciler.Log)
	assert.Nil(t, err)

	enabled, err := reconciler.ReconcileWorkflowController(dspa, params)
	assert.Nil(t, err)
	assert.True(t, enabled)

	policy := &admissionregistrationv1.ValidatingAdmissionPolicy{}
	err = reconciler.Get(ctx, types.NamespacedName{Name: "ds-pipeline-workflow-policy-" + testNamespace + "-" + testDSPAName}, policy)
	assert.Nil(t, err)

	role := &rbacv1.Role{}
	err = reconciler.Get(ctx, types.NamespacedName{
		Name:      "ds-pipeline-workflow-controller-role-" + testDSPAName,
		Namespace: testNamespace,
	}, role)
	assert.Nil(t, err)

	for _, rule := range role.Rules {
		assert.NotContains(t, rule.Resources, "pods/exec")
	}
}

func TestReconcileWorkflowControllerDisablesAndDeletesAdmissionResources(t *testing.T) {
	testNamespace := "testnamespace"
	testDSPAName := "testdspa"

	dspa := newWorkflowAdmissionTestDSPA(testNamespace, testDSPAName, true)
	ctx, params, reconciler := CreateNewTestObjects()

	err := params.ExtractParams(ctx, dspa, reconciler.Client, reconciler.Log)
	assert.Nil(t, err)

	enabled, err := reconciler.ReconcileWorkflowController(dspa, params)
	assert.Nil(t, err)
	assert.True(t, enabled)

	dspa.Spec.WorkflowController.Deploy = false
	enabled, err = reconciler.ReconcileWorkflowController(dspa, params)
	assert.Nil(t, err)
	assert.False(t, enabled)

	policy := &admissionregistrationv1.ValidatingAdmissionPolicy{}
	err = reconciler.Get(ctx, types.NamespacedName{Name: "ds-pipeline-workflow-policy-" + testNamespace + "-" + testDSPAName}, policy)
	assert.True(t, apierrs.IsNotFound(err))

	binding := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
	err = reconciler.Get(ctx, types.NamespacedName{Name: "ds-pipeline-workflow-policy-binding-" + testNamespace + "-" + testDSPAName}, binding)
	assert.True(t, apierrs.IsNotFound(err))
}

func TestWorkflowAdmissionPolicyContainsDenyRulesForUnsafeWorkflowFields(t *testing.T) {
	testNamespace := "testnamespace"
	testDSPAName := "testdspa"

	dspa := newWorkflowAdmissionTestDSPA(testNamespace, testDSPAName, true)
	ctx, params, reconciler := CreateNewTestObjects()

	err := params.ExtractParams(ctx, dspa, reconciler.Client, reconciler.Log)
	assert.Nil(t, err)

	err = reconciler.ReconcileWorkflowAdmission(dspa, params)
	assert.Nil(t, err)

	policy := &admissionregistrationv1.ValidatingAdmissionPolicy{}
	err = reconciler.Get(ctx, types.NamespacedName{Name: "ds-pipeline-workflow-policy-" + testNamespace + "-" + testDSPAName}, policy)
	assert.Nil(t, err)

	expressions := make([]string, 0, len(policy.Spec.Validations))
	messages := make([]string, 0, len(policy.Spec.Validations))
	for _, validation := range policy.Spec.Validations {
		expressions = append(expressions, validation.Expression)
		messages = append(messages, validation.Message)
	}
	allExpressions := strings.Join(expressions, "\n")
	allMessages := strings.Join(messages, "\n")

	// These checks verify the policy carries deny logic for the required unsafe fields.
	assert.Contains(t, allExpressions, "hostNetwork")
	assert.Contains(t, allExpressions, "podSpecPatch")
	assert.Contains(t, allExpressions, "hostpath")
	assert.Contains(t, allExpressions, params.PipelineRunnerServiceAccountName)

	assert.Contains(t, allMessages, "hostNetwork=true is not allowed")
	assert.Contains(t, allMessages, "podSpecPatch contains disallowed privileged or host-level settings")
	assert.Contains(t, allMessages, "serviceAccountName must be set to "+params.PipelineRunnerServiceAccountName)
	assert.Contains(t, allMessages, "hostPath volumes are not allowed")
}
