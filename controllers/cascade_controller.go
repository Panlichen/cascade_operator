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
	"context"
	goerrors "errors"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	derechov1alpha1 "github.com/Panlichen/cascade_operator/api/v1alpha1"
)

// CascadeReconciler reconciles a cascade object
type CascadeReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	NodeManagerMap map[string]*derechov1alpha1.CascadeNodeManager
}

const (
	selectorKey      string = "cascadeName"
	cascadeFinalizer string = "derecho.poanpan"
)

var (
	serverContainers []v1.Container = []v1.Container{{
		Image: "poanpan/cascade:upgrade-cascade-gpu",
		Name:  "server",
		// TODO: change command to start server
		Command: []string{"sh", "-c", "/usr/sbin/sshd && echo I am server && sleep 2592000"},
		Resources: v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceCPU:         resource.MustParse("2"),
				v1.ResourceMemory:      resource.MustParse("8Gi"),
				"openshift.io/mlx5_vf": resource.MustParse("1"),
			},
			Limits: v1.ResourceList{
				v1.ResourceCPU:         resource.MustParse("10"),
				v1.ResourceMemory:      resource.MustParse("20Gi"),
				"openshift.io/mlx5_vf": resource.MustParse("1")},
		},
	}}
	clientContainers []v1.Container = []v1.Container{{
		// TODO: need a light client image
		Image: "poanpan/cascade:upgrade-cascade-gpu",
		Name:  "client",
		// TODO: change command to start client
		Command: []string{"sh", "-c", "/usr/sbin/sshd && echo I am client && sleep 2592000"},
		Resources: v1.ResourceRequirements{
			// TODO: client may need less resource
			Requests: v1.ResourceList{
				v1.ResourceCPU:         resource.MustParse("2"),
				v1.ResourceMemory:      resource.MustParse("8Gi"),
				"openshift.io/mlx5_vf": resource.MustParse("1"),
			},
			Limits: v1.ResourceList{
				v1.ResourceCPU:         resource.MustParse("10"),
				v1.ResourceMemory:      resource.MustParse("20Gi"),
				"openshift.io/mlx5_vf": resource.MustParse("1")},
		},
	}}
)

//+kubebuilder:rbac:groups=derecho.poanpan,resources=cascades,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=derecho.poanpan,resources=cascades/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=derecho.poanpan,resources=cascades/finalizers,verbs=update
//+kubebuilder:rbac:groups=derecho.poanpan,resources=cascadenodemanagers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=derecho.poanpan,resources=cascadenodemanagers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=derecho.poanpan,resources=cascadenodemanagers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

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
	log.Info("\n\n\n\t\t*** Entering Reconile Logic ***\n\n")
	log.Info(fmt.Sprintf("Get request: %+v", req.NamespacedName))
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

	// !!!TODO: delete
	// Check if the Cascade instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isCascadeMarkedToBeDeleted := cascade.GetDeletionTimestamp() != nil
	if isCascadeMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(cascade, cascadeFinalizer) {
			// Run finalization logic for cascadeFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeCascade(ctx, log, cascade); err != nil {
				return ctrl.Result{}, err
			}
			// Remove cascadeFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(cascade, cascadeFinalizer)
			err := r.Update(ctx, cascade)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(cascade, cascadeFinalizer) {
		controllerutil.AddFinalizer(cascade, cascadeFinalizer)
		err = r.Update(ctx, cascade)
		if err != nil {
			log.Error(err, fmt.Sprintf("Add Finalizer for cascade %v failed", req.NamespacedName))
			return ctrl.Result{}, err
		}
	}

	if cascade.Status.LogicalServerSize == 0 {
		// this means we are creating the Cascade for the first time, we need to create NodeManager structure manually.
		// TODO: create the CascadeNodeManager CR, from tutorial: Note The Node field is just to illustrate an example of a Status field. In real cases, it would be recommended to use [Conditions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties).

		// Parse the configMap, create pods and the headless service, allocate memory for CascadeReconciler.NodeManager
		r.NodeManagerMap[req.Name] = new(derechov1alpha1.CascadeNodeManager)
		configMapFinder := cascade.Spec.ConfigMapFinder.DeepCopy()
		err = r.createNodeManager(ctx, log, req.NamespacedName, configMapFinder)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Check if the user request enough logical nodes to satisfy the Cascade's need according to the json layout file
		checkStat, checkErr := r.checkLogicalNodesRequest(cascade)
		if !checkStat {
			log.Error(checkErr, "Has Not Requested enough")
			return ctrl.Result{}, checkErr
		}
	}

	// var headlessService *v1.Service
	// NOTE: cannot just declare a pointer, but need a real pointer for r.Get
	headlessService := &v1.Service{}
	err = r.Get(ctx, req.NamespacedName, headlessService)
	if err != nil && errors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Create the Headless Service Cascade %v", cascade.Name))
		r.createHeadlessService(ctx, log, cascade)
	}

	// TODO: Currently only support adding an arbitraary number of nodes, or delete the whole Cascade. Deleting an arbitrary number of nodes requires collaboration with the layout information in the Cascade application.

	// Compare specified logical server number(cascade.Spec.LogicalServerSize) with current logical server number(cascade.Status.LogicalServerSize), and create miss pods.
	// TODO: what if we add more nodes than the sum of MaxNodes from all shards?
	specLogicalServerSize := cascade.Spec.LogicalServerSize
	statusLogicalServerSize := cascade.Status.LogicalServerSize

	log.Info(fmt.Sprintf("specLogicalServerSize is %v, statusLogicalServerSize is %v", specLogicalServerSize, statusLogicalServerSize))

	if statusLogicalServerSize < specLogicalServerSize {
		err = r.createPods(ctx, log, specLogicalServerSize-statusLogicalServerSize, cascade, true)
		if err != nil {
			log.Error(err, "Fail to create Server Pods")
		}
	}

	// Compare specified logical client number(cascade.Spec.ClientSize) with current logical client number(cascade.Status.ClientSize), and create miss pods.
	specClientSize := cascade.Spec.ClientSize
	statusClientSize := cascade.Status.ClientSize

	log.Info(fmt.Sprintf("specClientSize is %v, statusClientSize is %v", specClientSize, statusClientSize))

	if statusClientSize < specClientSize {
		err = r.createPods(ctx, log, specClientSize-statusClientSize, cascade, false)
		if err != nil {
			log.Error(err, "Fail to create Server Pods")
		}
	}

	return ctrl.Result{}, nil
}

func (r *CascadeReconciler) finalizeCascade(ctx context.Context, log logr.Logger, cascade *derechov1alpha1.Cascade) error {
	log.Info(fmt.Sprintf("Finalize deleted Cascade: %v", cascade.Name))
	// delete pods
	for _, podName := range cascade.Status.Nodes {
		podToDelete := &v1.Pod{}
		namespacedName := types.NamespacedName{Name: podName, Namespace: cascade.Namespace}
		err := r.Get(ctx, namespacedName, podToDelete)
		if err != nil {
			log.Error(err, fmt.Sprintf("Fail to get the pod to delete %v", namespacedName))
			return err
		}
		// TODO: maybe parallel in goroutine?
		err = r.Delete(ctx, podToDelete)
		if err != nil {
			log.Error(err, fmt.Sprintf("Fail to delete the pod to delete %v", namespacedName))
			return err
		}
		log.Info(fmt.Sprintf("Delete pod %v", namespacedName))
	}
	headlessServiceToDelete := &v1.Service{}
	namespacedName := types.NamespacedName{Name: cascade.Name, Namespace: cascade.Namespace}
	err := r.Get(ctx, namespacedName, headlessServiceToDelete)
	if err != nil {
		log.Error(err, fmt.Sprintf("Fail to get the headless service to delete %v", namespacedName))
	}
	err = r.Delete(ctx, headlessServiceToDelete)
	if err != nil {
		log.Error(err, fmt.Sprintf("Fail to delete the headless service to delete %v", namespacedName))
		return err
	}
	log.Info(fmt.Sprintf("Delete headless Service %v", namespacedName))

	return nil
}

func (r *CascadeReconciler) createHeadlessService(ctx context.Context, log logr.Logger, cascade *derechov1alpha1.Cascade) error {
	// create a headless service to provide fqdn
	serviceSelector := make(map[string]string)
	serviceSelector[selectorKey] = cascade.Name
	headlessService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cascade.Name,
			Namespace: cascade.Namespace,
			Labels:    serviceSelector,
		},
		Spec: v1.ServiceSpec{
			Selector:  serviceSelector,
			ClusterIP: "None",
		},
	}

	err := r.Create(ctx, headlessService)
	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to create headless service %v/%v", cascade.Namespace, cascade.Name))
		return err
	}
	log.Info(fmt.Sprintf("Create a headless service for Cascade %v successfully.", cascade.Name))

	return nil
}

func (r *CascadeReconciler) createPods(ctx context.Context, log logr.Logger, createCnt int, cascade *derechov1alpha1.Cascade, isServer bool) error {
	// TODO: here we use the un-reserved nodes directly. After we determine how to use reserved and overlapped node ids, we need to redesign this function
	nodeId := r.NodeManagerMap[cascade.Name].Status.NextNodeIdToAssign
	serviceSelector := make(map[string]string)
	serviceSelector[selectorKey] = cascade.Name

	// create pods as needed
	for i := 0; i < createCnt; i++ {
		var podName string

		var podSpec v1.PodSpec
		if isServer {
			podName = cascade.Name + "-server-" + fmt.Sprint(nodeId)
			podSpec = v1.PodSpec{
				Hostname:   podName,
				Subdomain:  cascade.Name,
				Containers: serverContainers,
				// the default container RestartPolicy is Always, very well.
			}
		} else {
			podName = cascade.Name + "-client-" + fmt.Sprint(nodeId)
			podSpec = v1.PodSpec{
				Hostname:   podName,
				Subdomain:  cascade.Name,
				Containers: clientContainers,
				// the default container RestartPolicy is Always, very well.
			}
		}
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: cascade.Namespace,
				Labels:    serviceSelector,
			},
			Spec: podSpec,
		}
		// update the temporary variable, not to update r.NodeManagerMap[cascade.Name].Status.NextNodeIdToAssign until all pods are created successfully
		nodeId++
		log.Info(fmt.Sprintf("Prepare to create pod %v", podName))
		err := r.Create(ctx, pod)
		if err != nil {
			log.Error(err, fmt.Sprintf("Failed to create pod %v/%v", cascade.Namespace, podName))
			return err
		}
		log.Info(fmt.Sprintf("Create pod %v for Cascade %v successfully", podName, cascade.Name))
	}

	// update cascade.Status.LogicalServerSize
	if isServer {
		// TODO: consider update cascade.Status.PhysicalServerSize
		cascade.Status.LogicalServerSize = createCnt
		err := r.Status().Update(ctx, cascade)
		if err != nil {
			log.Error(err, "Failed to update cascade status")
			return err
		}
	} else {
		cascade.Status.ClientSize = createCnt
		err := r.Status().Update(ctx, cascade)
		if err != nil {
			log.Error(err, "Failed to update cascade status")
			return err
		}
	}

	// Update the cascade status with the pod names
	podList := &v1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(cascade.Namespace),
		client.MatchingLabels(labelsForCascade(cascade.Name)),
	}
	if err := r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "cascade.Namespace", cascade.Namespace, "cascade.Name", cascade.Name)
		return err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, cascade.Status.Nodes) {
		cascade.Status.Nodes = podNames
	}
	err := r.Status().Update(ctx, cascade)
	if err != nil {
		log.Error(err, "Failed to update cascade status")
		return err
	}

	// after assign nodes, update NextNodeIdToAssign status for current Cascade.
	r.NodeManagerMap[cascade.Name].Status.NextNodeIdToAssign = nodeId

	return nil
}

func (r *CascadeReconciler) checkLogicalNodesRequest(cascade *derechov1alpha1.Cascade) (bool, error) {
	if cascade.Spec.LogicalServerSize >= r.NodeManagerMap[cascade.Name].Status.LeastRequiredLogicalNodes {
		return true, nil
	} else {
		return false, goerrors.New("not request enough logical nodes")
	}
}

// createNodeManager parse the json from the configMap user defined
func (r *CascadeReconciler) createNodeManager(ctx context.Context, log logr.Logger, cascadeInfo types.NamespacedName, configMapFinder *derechov1alpha1.CascadeConfigMapFinder) error {
	realConfigMap := &v1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Namespace: cascadeInfo.Namespace, Name: configMapFinder.Name}, realConfigMap)
	if err != nil {
		// if cannot find the config map, we cannot continue create other resources for Cascade
		log.Error(err, fmt.Sprintf("Fail to get the ConfigMap %v/%v", cascadeInfo.Namespace, configMapFinder.Name))
		return err
	}
	jsonStr := realConfigMap.Data[configMapFinder.JsonItem]
	log.Info(fmt.Sprintf("Get the jsonStr with length %v", len(jsonStr)))
	jsonStr = "{\"typesSpec\": " + jsonStr + "}"
	log.Info(fmt.Sprintf("Get the patched jsonStr with length %v", len(jsonStr)))
	log.Info(fmt.Sprintf("The patched jsonStr is: %v", jsonStr))

	// Allocate Memory
	r.NodeManagerMap[cascadeInfo.Name] = new(derechov1alpha1.CascadeNodeManager)

	log.Info(fmt.Sprintf("Create an entry in NodeManagerMap for new Cascade: %+v", cascadeInfo))

	json.Unmarshal([]byte(jsonStr), &r.NodeManagerMap[cascadeInfo.Name].Spec)
	log.Info(fmt.Sprintf("Unmarshal done, parse %v types", len(r.NodeManagerMap[cascadeInfo.Name].Spec.TypesSpec)))
	for seq, cascadeType := range r.NodeManagerMap[cascadeInfo.Name].Spec.TypesSpec {
		log.Info(fmt.Sprintf("Type %v has configuration %v", seq, cascadeType.String()))
	}

	r.NodeManagerMap[cascadeInfo.Name].Status.TypesStatus = r.NodeManagerMap[cascadeInfo.Name].Spec.DeepCopy().TypesSpec

	maxReservedNodeId := -1
	leastRequiredLogicalNodes := 0
	maxLogicalNodes := 0
	for typeSeq, cascadeType := range r.NodeManagerMap[cascadeInfo.Name].Status.TypesStatus {
		for subgroupSeq, subgroupLayout := range cascadeType.SubgroupLayout {
			shardNum := len(subgroupLayout.MinNodesByShard)
			log.Info(fmt.Sprintf("For type #%v(%v), subgroup #%v, shard num is %v",
				typeSeq, cascadeType.TypeAlias, subgroupSeq, shardNum))
			subgroupLayout.AssignedNodeIdByShard = make([][]int, shardNum)

			// if the user assigned reserved node_ids
			if len(subgroupLayout.ReservedNodeIdByShard) == shardNum {
				for shardSeq, reservedNodeIds := range subgroupLayout.ReservedNodeIdByShard {
					log.Info(fmt.Sprintf("For type #%v(%v), subgroup #%v, shard #%v reserves %v nodes: %+v",
						typeSeq, cascadeType.TypeAlias, subgroupSeq, shardSeq, len(reservedNodeIds), reservedNodeIds))

					for _, reservedNodeId := range reservedNodeIds {
						if maxReservedNodeId <= reservedNodeId {
							maxReservedNodeId = reservedNodeId
						}
					}
				}
			}

			for _, min_nodes := range subgroupLayout.MinNodesByShard {
				leastRequiredLogicalNodes += min_nodes
			}
			for _, max_nodes := range subgroupLayout.MaxNodesByShard {
				maxLogicalNodes += max_nodes
			}
		}
	}

	r.NodeManagerMap[cascadeInfo.Name].Status.MaxReservedNodeId = maxReservedNodeId
	r.NodeManagerMap[cascadeInfo.Name].Status.NextNodeIdToAssign = maxReservedNodeId + 1
	r.NodeManagerMap[cascadeInfo.Name].Status.LeastRequiredLogicalNodes = leastRequiredLogicalNodes
	r.NodeManagerMap[cascadeInfo.Name].Status.MaxLogicalNodes = maxLogicalNodes
	log.Info(fmt.Sprintf("For Cascade %+v, max reserved node id is %v, next node id to assign is %v, it needs at least %v logical nodes", cascadeInfo,
		r.NodeManagerMap[cascadeInfo.Name].Status.MaxReservedNodeId,
		r.NodeManagerMap[cascadeInfo.Name].Status.NextNodeIdToAssign,
		r.NodeManagerMap[cascadeInfo.Name].Status.LeastRequiredLogicalNodes))
	return nil
}

// labelsForCascade returns the labels for selecting the resources
// belonging to the given cascade CR name.
func labelsForCascade(name string) map[string]string {
	return map[string]string{selectorKey: name}
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

	r.NodeManagerMap = make(map[string]*derechov1alpha1.CascadeNodeManager)

	return ctrl.NewControllerManagedBy(mgr).
		For(&derechov1alpha1.Cascade{}).
		Owns(&v1.Pod{}).
		Owns(&v1.Service{}).
		Complete(r)
}
