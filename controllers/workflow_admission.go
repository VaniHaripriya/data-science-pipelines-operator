/*
Copyright 2026.

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
	"fmt"

	dspav1 "github.com/opendatahub-io/data-science-pipelines-operator/api/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
)

var workflowAdmissionTemplatesDir = "workflow-admission"

var workflowAdmissionTemplates = []string{
	"validating_admission_policy.yaml.tmpl",
	"validating_admission_policy_binding.yaml.tmpl",
}

// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingadmissionpolicies;validatingadmissionpolicybindings,verbs=create;get;update;patch;delete

func (r *DSPAReconciler) ReconcileWorkflowAdmission(dsp *dspav1.DataSciencePipelinesApplication,
	params *DSPAParams) error {

	log := r.Log.WithValues("namespace", dsp.Namespace).WithValues("dspa_name", dsp.Name)
	log.Info("Applying Workflow Admission Resources")

	for _, template := range workflowAdmissionTemplates {
		if err := r.ApplyWithoutOwner(params, workflowAdmissionTemplatesDir+"/"+template); err != nil {
			if apimeta.IsNoMatchError(err) {
				return fmt.Errorf("workflow admission requires ValidatingAdmissionPolicy support (admissionregistration.k8s.io/v1), which needs Kubernetes >=1.30 (OpenShift >=4.17). Ensure the cluster version meets this requirement: %w", err)
			}
			return err
		}
	}

	log.Info("Finished applying Workflow Admission Resources")
	return nil
}

func (r *DSPAReconciler) DeleteWorkflowAdmission(params *DSPAParams) error {
	for _, template := range workflowAdmissionTemplates {
		if err := r.DeleteResource(params, workflowAdmissionTemplatesDir+"/"+template); err != nil {
			if !apierrs.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}
