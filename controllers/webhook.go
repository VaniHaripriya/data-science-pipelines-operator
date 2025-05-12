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
	"context"
	"strings"

	dspav1 "github.com/opendatahub-io/data-science-pipelines-operator/api/v1"
	admv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var webhookTemplatesDir = "webhook/"

const operatorName = "data-science-pipelines-operator-controller-manager"

//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete

func (r *DSPAReconciler) ReconcileWebhook(ctx context.Context, dsp *dspav1.DataSciencePipelinesApplication, params *DSPAParams) error {
	log := r.Log.WithValues("namespace", params.DSPONamespace).WithValues("dspa_name", dsp.Name)

	if !dsp.Spec.APIServer.Deploy {
		r.Log.Info("Skipping Application of Webhook Resources")
		return nil
	}

	dataSciencePipelinesOperator := appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Namespace: params.DSPONamespace, Name: operatorName}, &dataSciencePipelinesOperator)
	if err != nil {
		return err
	}

	log.Info("Applying Webhook Resources")
	err = r.ApplyDir(&dataSciencePipelinesOperator, params, webhookTemplatesDir)
	if err != nil {
		return err
	}

	log.Info("Finished applying Webhook Resources")
	return nil
}

func (r *DSPAReconciler) CleanWebhook(ctx context.Context, dspa *dspav1.DataSciencePipelinesApplication, params *DSPAParams) error {
	log := r.Log.WithValues("namespace", params.DSPONamespace, "dspa_name", dspa.Name)

	hasK8sDSPAs, err := r.checkAvailableKubernetesDSPAs(ctx, dspa.Name, dspa.Namespace)
	if err != nil {
		log.Error(err, "Failed to check for other DSPAs with 'kubernetes' storage")
		return err
	}

	if !hasK8sDSPAs {
		log.Info("No other DSPAs with PipelineStorage 'kubernetes' found. Cleaning up webhook resources.")
		if err := r.cleanupWebhookResources(ctx, params.DSPONamespace); err != nil {
			log.Error(err, "Failed to clean up webhook resources")
			return err
		}
	}

	log.Info("Webhook resources cleanup complete.")
	return nil
}

func (r *DSPAReconciler) cleanupWebhookResources(ctx context.Context, namespace string) error {
	log := r.Log.WithValues("namespace", namespace)
	webhookName := "pipelineversions.pipelines.kubeflow.org"
	k8sWebhookName := "kubernetes-webhook"
	webhookNamespaced := types.NamespacedName{Name: k8sWebhookName, Namespace: namespace}

	// Delete MutatingWebhookConfiguration
	mutating := &admv1.MutatingWebhookConfiguration{}
	if err := r.Get(ctx, types.NamespacedName{Name: webhookName}, mutating); err == nil {
		log.Info("Deleting MutatingWebhookConfiguration", "webhook", mutating)
		if err := r.Delete(ctx, mutating); err != nil {
			log.Error(err, "Failed to delete MutatingWebhookConfiguration")
			return err
		}
	}

	// Delete ValidatingWebhookConfiguration
	validating := &admv1.ValidatingWebhookConfiguration{}
	if err := r.Get(ctx, types.NamespacedName{Name: webhookName}, validating); err == nil {
		log.Info("Deleting ValidatingWebhookConfiguration")
		if err := r.Delete(ctx, validating); err != nil {
			log.Error(err, "Failed to delete ValidatingWebhookConfiguration")
			return err
		}
	}

	// Delete Role
	role := &rbacv1.ClusterRole{}
	if err := r.Get(ctx, types.NamespacedName{Name: k8sWebhookName}, role); err == nil {
		log.Info("Deleting webhook Role", "role", role)
		if err := r.Delete(ctx, role); err != nil {
			log.Error(err, "Failed to delete webhook Role")
			return err
		}
	}

	// Delete RoleBinding
	roleBinding := &rbacv1.ClusterRoleBinding{}
	if err := r.Get(ctx, types.NamespacedName{Name: k8sWebhookName}, roleBinding); err == nil {
		log.Info("Deleting webhook RoleBinding", "roleBinding", roleBinding)
		if err := r.Delete(ctx, roleBinding); err != nil {
			log.Error(err, "Failed to delete webhook RoleBinding")
			return err
		}
	}

	// Delete Service
	svc := &corev1.Service{}
	if err := r.Get(ctx, webhookNamespaced, svc); err == nil {
		log.Info("Deleting webhook Service")
		if err := r.Delete(ctx, svc); err != nil {
			log.Error(err, "Failed to delete webhook Service")
			return err
		}
	}

	// Delete ServiceAccount
	sa := &corev1.ServiceAccount{}
	if err := r.Get(ctx, webhookNamespaced, sa); err == nil {
		log.Info("Deleting webhook ServiceAccount")
		if err := r.Delete(ctx, sa); err != nil {
			log.Error(err, "Failed to delete webhook ServiceAccount")
			return err
		}
	}

	// Delete deployments
	var deployList appsv1.DeploymentList
	if err := r.List(ctx, &deployList, &client.ListOptions{Namespace: namespace}); err != nil {
		log.Error(err, "Failed to list Deployments")
		return err
	}
	for _, deploy := range deployList.Items {
		if strings.HasPrefix(deploy.Name, k8sWebhookName) {
			log.Info("Deleting webhook Deployment", "name", deploy.Name)
			if err := r.Delete(ctx, &deploy); err != nil {
				log.Error(err, "Failed to delete webhook Deployment", "name", deploy.Name)
				return err
			}
		}
	}

	return nil
}
