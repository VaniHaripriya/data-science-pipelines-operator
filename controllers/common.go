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

	dspav1 "github.com/opendatahub-io/data-science-pipelines-operator/api/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var commonTemplatesDir = "common/default"

const commonCusterRolebindingTemplate = "common/no-owner/clusterrolebinding.yaml.tmpl"

var legacyArgoAggregateClusterRoles = []string{
	"argo-aggregate-to-admin",
	"argo-aggregate-to-edit",
	"argo-aggregate-to-view",
}

func (r *DSPAReconciler) ReconcileCommon(dsp *dspav1.DataSciencePipelinesApplication, params *DSPAParams) error {
	log := r.Log.WithValues("namespace", dsp.Namespace).WithValues("dspa_name", dsp.Name)

	log.Info("Applying Common Resources")
	err := r.ApplyDir(dsp, params, commonTemplatesDir)
	if err != nil {
		return err
	}
	err = r.ApplyWithoutOwner(params, commonCusterRolebindingTemplate)
	if err != nil {
		return err
	}
	if err = r.DeleteLegacyArgoAggregateClusterRoles(context.Background()); err != nil {
		return err
	}

	log.Info("Finished applying Common Resources")
	return nil
}

func (r *DSPAReconciler) CleanUpCommon(params *DSPAParams) error {
	err := r.DeleteResource(params, commonCusterRolebindingTemplate)
	if err != nil {
		return err
	}
	return nil
}

func (r *DSPAReconciler) DeleteLegacyArgoAggregateClusterRoles(ctx context.Context) error {
	for _, roleName := range legacyArgoAggregateClusterRoles {
		role := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: roleName},
		}
		if err := r.Delete(ctx, role); err != nil && !apierrs.IsNotFound(err) {
			return err
		}
	}
	return nil
}
