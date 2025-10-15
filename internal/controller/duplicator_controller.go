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
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	syozev1 "syoze.fr/duplicator/api/v1"
)

// DuplicatorReconciler reconciles a Duplicator object
type DuplicatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=syoze.syoze.fr,resources=duplicators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=syoze.syoze.fr,resources=duplicators/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=syoze.syoze.fr,resources=duplicators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Duplicator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *DuplicatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	logger.Info("Reconciling Duplicator", "name", req.Name, "namespace", req.Namespace)

	var duplicator syozev1.Duplicator
	if err := r.Get(ctx, req.NamespacedName, &duplicator); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	finalizerName := "syoze.syoze.fr/duplicator-finalizer"
	// Add finalizer if not present and not being deleted
	if duplicator.ObjectMeta.DeletionTimestamp == nil && !containsString(duplicator.Finalizers, finalizerName) {
		duplicator.Finalizers = append(duplicator.Finalizers, finalizerName)
		if err := r.Update(ctx, &duplicator); err != nil {
			logger.Error(err, "unable to add finalizer")
			return ctrl.Result{}, err
		}
	}

	var nsList v1.NamespaceList
	if err := r.List(ctx, &nsList, client.MatchingLabels(duplicator.Spec.NamespaceSelector.MatchLabels)); err != nil {
		logger.Error(err, "unable to list namespaces")
		return ctrl.Result{}, err
	}

	if duplicator.ObjectMeta.DeletionTimestamp != nil {
		// Handle deletion: remove duplicated resources in target namespace
		for _, ns := range nsList.Items {
			for _, tr := range duplicator.Spec.TargetResources {
				switch tr.Kind {
				case "ConfigMap":
					if err := r.Delete(ctx, &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: ns.Name, Name: tr.Name}}); err != nil {
						return ctrl.Result{}, fmt.Errorf("unable to delete ConfigMap: %w", err)
					} else {
						logger.Info("Deleted ConfigMap from target namespace", "name", tr.Name, "namespace", duplicator.Namespace)
					}
				case "Secret":
					if err := r.Delete(ctx, &v1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: ns.Name, Name: tr.Name}}); err != nil {
						return ctrl.Result{}, fmt.Errorf("unable to delete Secret: %w", err)
					} else {
						logger.Info("Deleted Secret from target namespace", "name", tr.Name, "namespace", duplicator.Namespace)
					}
				}
			}
		}

		// Remove finalizer
		if containsString(duplicator.Finalizers, finalizerName) {
			duplicator.Finalizers = removeString(duplicator.Finalizers, finalizerName)
			if err := r.Update(ctx, &duplicator); err != nil {
				logger.Error(err, "unable to remove finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	for _, ns := range nsList.Items {
		logger.Info("Matched namespace", "name", ns.Name)
		for _, tr := range duplicator.Spec.TargetResources {
			switch tr.Kind {
			case "ConfigMap":
				var cm v1.ConfigMap
				if err := r.Client.Get(ctx, client.ObjectKey{Namespace: tr.Namespace, Name: tr.Name}, &cm); err != nil {
					return ctrl.Result{}, fmt.Errorf("unable to fetch Secret %s/%s: %w", tr.Namespace, tr.Name, err)
				}

				if err := duplicateAndCreateObject(ctx, r.Client, &cm, ns.Name, &duplicator); err != nil {
					return ctrl.Result{}, err
				}
			case "Secret":
				var secret v1.Secret
				if err := r.Client.Get(ctx, client.ObjectKey{Namespace: tr.Namespace, Name: tr.Name}, &secret); err != nil {
					return ctrl.Result{}, fmt.Errorf("unable to fetch Secret %s/%s: %w", tr.Namespace, tr.Name, err)
				}

				if err := duplicateAndCreateObject(ctx, r.Client, &secret, ns.Name, &duplicator); err != nil {
					return ctrl.Result{}, err
				}

			default:
				return ctrl.Result{}, fmt.Errorf("duplicator does not support this object")
			}
		}
	}

	return ctrl.Result{}, nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	result := []string{}
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

func duplicateObject(obj client.Object, targetNamespace string) client.Object {
	cp := obj.DeepCopyObject().(client.Object)
	cp.SetNamespace(targetNamespace)
	cp.SetResourceVersion("")
	cp.SetUID("")
	if metaObj, ok := cp.(metav1.Object); ok {
		metaObj.SetCreationTimestamp(metav1.Time{})
	}

	return cp
}

func duplicateAndCreateObject(ctx context.Context, cli client.Client, obj client.Object, targetNamespace string, owner client.Object) error {
	objectCopy := duplicateObject(obj, targetNamespace)

	if _, err := ctrl.CreateOrUpdate(ctx, cli, objectCopy, func() error {
		labels := objectCopy.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}

		labels["syoze.syoze.fr/managed"] = "true"
		labels["syoze.syoze.fr/managed-by-name"] = owner.GetName()
		labels["syoze.syoze.fr/managed-by-namespace"] = owner.GetNamespace()

		objectCopy.SetLabels(labels)

		return nil
	}); err != nil {
		return fmt.Errorf("failed to create object: %w", err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DuplicatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syozev1.Duplicator{}).
		Watches(
			&v1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				if obj.GetDeletionTimestamp() != nil {
					return []reconcile.Request{}
				}

				namespaceLabels := obj.GetLabels()

				var duplicators syozev1.DuplicatorList
				if err := r.List(ctx, &duplicators); err != nil {
					return []ctrl.Request{}
				}

				var reqs []ctrl.Request
				for _, d := range duplicators.Items {
					match := true
					for k, v := range d.Spec.NamespaceSelector.MatchLabels {
						if namespaceLabels[k] != v {
							match = false
						}
					}

					if match {
						reqs = append(reqs, ctrl.Request{
							NamespacedName: client.ObjectKey{
								Namespace: d.Namespace,
								Name:      d.Name,
							},
						})
					}
				}
				return reqs
			}),
		).
		Named("duplicator").
		Complete(r)
}
