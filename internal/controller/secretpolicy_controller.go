/*
Copyright 2025.

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

package controller

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	secretsv1alpha1 "github.com/miltlima/secrethor/api/v1alpha1"
)

// SecretPolicyReconciler reconciles a SecretPolicy object
type SecretPolicyReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=secrets.secrethor.dev,resources=secretpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=secrets.secrethor.dev,resources=secretpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=secrets.secrethor.dev,resources=secretpolicies/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *SecretPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("secretpolicy", req.NamespacedName)

	var policy secretsv1alpha1.SecretPolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("SecretPolicy not found, skipping reconciliation")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	log.Info("Reconciling SecretPolicy")

	var secrets corev1.SecretList
	if err := r.List(ctx, &secrets); err != nil {
		log.Error(err, "Failed to list secrets")
		return ctrl.Result{}, err
	}

	now := time.Now()
	for _, secret := range secrets.Items {
		age := now.Sub(secret.CreationTimestamp.Time).Hours() / 24
		allowed := false
		for _, ns := range policy.Spec.AllowedNamespaces {
			if secret.Namespace == ns {
				allowed = true
				break
			}
		}
		if !allowed {
			log.Info("Secret not in allowed namespace", "secret", secret.Name, "namespace", secret.Namespace)
		}
		if policy.Spec.MaxAgeDays > 0 && int(age) > policy.Spec.MaxAgeDays {
			log.Info("Secret expired ", "secret", secret.Name, "age", age)
		}
	}

	return ctrl.Result{RequeueAfter: 1 * time.Hour}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretsv1alpha1.SecretPolicy{}).
		Complete(r)
}
