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

package controllers

import (
	kapps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appv1 "github.com/anisurrahman75/Ex-M-R-ConfigDeployment/api/v1"
)

// ConfigDeploymentReconciler reconciles a ConfigDeployment object
type ConfigDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.tutorial.kubebuilder.io,resources=configdeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.tutorial.kubebuilder.io,resources=configdeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.tutorial.kubebuilder.io,resources=configdeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps.tutorial.kubebuilder.io,resources=configdeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.tutorial.kubebuilder.io,resources=configdeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.tutorial.kubebuilder.io,resources=configdeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ConfigDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ConfigDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	_ = log
	var configDeployment appv1.ConfigDeployment
	if err := r.Get(ctx, req.NamespacedName, &configDeployment); err != nil {
		log.Error(err, "unable to fetch ConfigDeployment")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	fmt.Println("Trying")
	var configMapVersion string
	if configDeployment.Spec.ConfigMap != "" {
		configMapName := configDeployment.Spec.ConfigMap
		foundConfigMap := &corev1.ConfigMap{}
		err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: configDeployment.Namespace}, foundConfigMap)
		if err != nil {
			// If a configMap name is provided, then it must exist
			// You will likely want to create an Event for the user to understand why their reconcile is failing.
			return ctrl.Result{}, err
		}

		// Hash the data in some way, or just use the version of the Object
		configMapVersion = foundConfigMap.ResourceVersion
	}
	fmt.Println(configMapVersion)
	return ctrl.Result{}, nil
}

const (
	configMapField = ".spec.configMap"
)

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appv1.ConfigDeployment{}, configMapField, func(rawObj client.Object) []string {
		// Extract the ConfigMap name from the ConfigDeployment Spec, if one is provided
		configDeployment := rawObj.(*appv1.ConfigDeployment)
		if configDeployment.Spec.ConfigMap == "" {
			return nil
		}
		return []string{configDeployment.Spec.ConfigMap}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.ConfigDeployment{}).
		Owns(&kapps.Deployment{}).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForConfigMap),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).Complete(r)
}
func (r *ConfigDeploymentReconciler) findObjectsForConfigMap(configMap client.Object) []reconcile.Request {
	attachedConfigDeployments := &appv1.ConfigDeploymentList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(configMapField, configMap.GetName()),
		Namespace:     configMap.GetNamespace(),
	}
	err := r.List(context.TODO(), attachedConfigDeployments, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(attachedConfigDeployments.Items))
	for i, item := range attachedConfigDeployments.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}
