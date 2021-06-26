/*
Copyright 2021.

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
	"reflect"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	derechov1alpha1 "github.com/Panlichen/cascade_operator/api/v1alpha1"
)

// CascadeReconciler reconciles a cascade object
type CascadeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=derecho.poanpan,resources=cascades,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=derecho.poanpan,resources=cascades/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=derecho.poanpan,resources=cascades/finalizers,verbs=update
//+kubebuilder:rbac:groups=derecho.poanpan,resources=cascadenodemanagers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=derecho.poanpan,resources=cascadenodemanagers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=derecho.poanpan,resources=cascadenodemanagers/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the cascade object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CascadeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	// Fetch the cascade instance
	cascade := &derechov1alpha1.Cascade{}
	err := r.Get(ctx, req.NamespacedName, cascade)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("cascade resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get cascade")
		return ctrl.Result{}, err
	}

	// Check if the headless service exists, which reflects the status of corresponding pods
	headless_service := &v1.Service{}
	err = r.Get(ctx, req.NamespacedName, headless_service)
	if err != nil && errors.IsNotFound(err) {
		// Parse the configMap, create pods and the headless service
	}

	// Update the cascade status with the pod names
	// List the pods for this cascade's deployment
	podList := &v1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(cascade.Namespace),
		client.MatchingLabels(labelsForCascade(cascade.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "cascade.Namespace", cascade.Namespace, "cascade.Name", cascade.Name)
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, cascade.Status.Nodes) {
		cascade.Status.Nodes = podNames
		err := r.Status().Update(ctx, cascade)
		if err != nil {
			log.Error(err, "Failed to update cascade status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// labelsForCascade returns the labels for selecting the resources
// belonging to the given cascade CR name.
func labelsForCascade(name string) map[string]string {
	return map[string]string{"app": "cascade", "cascade_cr": name}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []v1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

// SetupWithManager sets up the controller with the Manager.
func (r *CascadeReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// TODO: start prometheus and layout watcher here.

	return ctrl.NewControllerManagedBy(mgr).
		For(&derechov1alpha1.Cascade{}).
		Owns(&v1.Pod{}).
		Owns(&v1.Service{}).
		Complete(r)
}
