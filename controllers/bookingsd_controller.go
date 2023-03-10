/*
Copyright 2022.

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
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"os"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	bookingsv1alpha1 "github.com/calvarado2004/bookings-operator/api/v1alpha1"
)

const bookingsdFinalizer = "bookings.calvarado04.com/finalizer"

// Definitions to manage status conditions
const (
	// typeAvailableBookingsd represents the status of the Deployment reconciliation
	typeAvailableBookingsd = "Available"
	// typeDegradedBookingsd represents the status used when the custom resource is deleted and the finalizer operations are must to occur.
	typeDegradedBookingsd = "Degraded"
)

// BookingsdReconciler reconciles a Bookingsd object
type BookingsdReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// The following markers are used to generate the rules permissions (RBAC) on config/rbac using controller-gen
// when the command <make manifests> is executed.
// To know more about markers see: https://book.kubebuilder.io/reference/markers.html

//+kubebuilder:rbac:groups=bookings.calvarado04.com,resources=bookingsds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bookings.calvarado04.com,resources=bookingsds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=bookings.calvarado04.com,resources=bookingsds/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods;services;events;secrets;configmaps;persistentvolumeclaims,verbs=create;patch;update;delete;get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.

// It is essential for the controller's reconciliation loop to be idempotent. By following the Operator
// pattern you will create Controllers which provide a reconcile function
// responsible for synchronizing resources until the desired state is reached on the cluster.
// Breaking this recommendation goes against the design principles of controller-runtime.
// and may lead to unforeseen consequences such as resources becoming stuck and requiring manual intervention.
// For further info:
// - About Operator Pattern: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
// - About Controllers: https://kubernetes.io/docs/concepts/architecture/controller/
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *BookingsdReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Bookingsd instance
	// The purpose is check if the Custom Resource for the Kind Bookingsd
	// is applied on the cluster if not we return nil to stop the reconciliation
	Bookingsd := &bookingsv1alpha1.Bookingsd{}
	err := r.Get(ctx, req.NamespacedName, Bookingsd)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("Bookingsd resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Bookingsd")
		return ctrl.Result{}, err
	}

	// Let's just set the status as Unknown when no status are available
	if Bookingsd.Status.Conditions == nil || len(Bookingsd.Status.Conditions) == 0 {
		meta.SetStatusCondition(&Bookingsd.Status.Conditions, metav1.Condition{Type: typeAvailableBookingsd, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
		if err = r.Status().Update(ctx, Bookingsd); err != nil {
			log.Error(err, "Failed to update Bookingsd status")
			return ctrl.Result{}, err
		}

		// Let's re-fetch the Bookingsd Custom Resource after update the status
		// so that we have the latest state of the resource on the cluster and we will avoid
		// raise the issue "the object has been modified, please apply
		// your changes to the latest version and try again" which would re-trigger the reconciliation
		// if we try to update it again in the following operations
		if err := r.Get(ctx, req.NamespacedName, Bookingsd); err != nil {
			log.Error(err, "Failed to re-fetch Bookingsd")
			return ctrl.Result{}, err
		}
	}

	// Let's add a finalizer. Then, we can define some operations which should
	// occurs before the custom resource to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if !controllerutil.ContainsFinalizer(Bookingsd, bookingsdFinalizer) {
		log.Info("Adding Finalizer for Bookingsd")
		if ok := controllerutil.AddFinalizer(Bookingsd, bookingsdFinalizer); !ok {
			log.Error(err, "Failed to add finalizer into the custom resource")
			return ctrl.Result{Requeue: true}, nil
		}

		if err = r.Update(ctx, Bookingsd); err != nil {
			log.Error(err, "Failed to update custom resource to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Check if the Bookingsd instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isBookingsdMarkedToBeDeleted := Bookingsd.GetDeletionTimestamp() != nil
	if isBookingsdMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(Bookingsd, bookingsdFinalizer) {
			log.Info("Performing Finalizer Operations for Bookingsd before delete CR")

			// Let's add here an status "Downgrade" to define that this resource begin its process to be terminated.
			meta.SetStatusCondition(&Bookingsd.Status.Conditions, metav1.Condition{Type: typeDegradedBookingsd,
				Status: metav1.ConditionUnknown, Reason: "Finalizing",
				Message: fmt.Sprintf("Performing finalizer operations for the custom resource: %s ", Bookingsd.Name)})

			if err := r.Status().Update(ctx, Bookingsd); err != nil {
				log.Error(err, "Failed to update Bookingsd status")
				return ctrl.Result{}, err
			}

			// Perform all operations required before remove the finalizer and allow
			// the Kubernetes API to remove the custom resource.
			r.doFinalizerOperationsForBookingsd(Bookingsd)

			// TODO(user): If you add operations to the doFinalizerOperationsForBookingsd method
			// then you need to ensure that all worked fine before deleting and updating the Downgrade status
			// otherwise, you should requeue here.

			// Re-fetch the Bookingsd Custom Resource before update the status
			// so that we have the latest state of the resource on the cluster and we will avoid
			// raise the issue "the object has been modified, please apply
			// your changes to the latest version and try again" which would re-trigger the reconciliation
			if err := r.Get(ctx, req.NamespacedName, Bookingsd); err != nil {
				log.Error(err, "Failed to re-fetch Bookingsd")
				return ctrl.Result{}, err
			}

			meta.SetStatusCondition(&Bookingsd.Status.Conditions, metav1.Condition{Type: typeDegradedBookingsd,
				Status: metav1.ConditionTrue, Reason: "Finalizing",
				Message: fmt.Sprintf("Finalizer operations for custom resource %s name were successfully accomplished", Bookingsd.Name)})

			if err := r.Status().Update(ctx, Bookingsd); err != nil {
				log.Error(err, "Failed to update Bookingsd status")
				return ctrl.Result{}, err
			}

			log.Info("Removing Finalizer for Bookingsd after successfully perform the operations")
			if ok := controllerutil.RemoveFinalizer(Bookingsd, bookingsdFinalizer); !ok {
				log.Error(err, "Failed to remove finalizer for Bookingsd")
				return ctrl.Result{Requeue: true}, nil
			}

			if err := r.Update(ctx, Bookingsd); err != nil {
				log.Error(err, "Failed to remove finalizer for Bookingsd")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Check if service already exists, if not create a new one
	foundService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: "bookings-svc", Namespace: Bookingsd.Namespace}, foundService)
	if err != nil && apierrors.IsNotFound(err) {

		svc, err := r.serviceForBookings(Bookingsd)
		if err != nil {
			log.Error(err, "Failed to define new Service resource for Bookingsd")

			// The following implementation will update the status
			meta.SetStatusCondition(&Bookingsd.Status.Conditions, metav1.Condition{Type: typeAvailableBookingsd,
				Status: metav1.ConditionFalse, Reason: "Reconciling",
				Message: fmt.Sprintf("Failed to create Service for the custom resource (%s): (%s)", Bookingsd.Name, err)})

			if err := r.Status().Update(ctx, Bookingsd); err != nil {
				log.Error(err, "Failed to update Bookingsd status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		log.Info("Creating a new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		if err = r.Create(ctx, svc); err != nil {
			log.Error(err, "Failed to create new Service",
				"Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return ctrl.Result{}, err
		}

		// Service created successfully
		// We will requeue the reconciliation so that we can ensure the state
		// and move forward for the next operations

		return ctrl.Result{RequeueAfter: time.Minute}, nil

	} else if err != nil {
		log.Error(err, "Failed to get Service")
		// Let's return the error for the reconciliation be re-trigged again
		return ctrl.Result{}, err
	}

	// Check if deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: Bookingsd.Name, Namespace: Bookingsd.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new deployment
		dep, err := r.deploymentForBookingsd(Bookingsd)
		if err != nil {
			log.Error(err, "Failed to define new Deployment resource for Bookingsd")

			// The following implementation will update the status
			meta.SetStatusCondition(&Bookingsd.Status.Conditions, metav1.Condition{Type: typeAvailableBookingsd,
				Status: metav1.ConditionFalse, Reason: "Reconciling",
				Message: fmt.Sprintf("Failed to create Deployment for the custom resource (%s): (%s)", Bookingsd.Name, err)})

			if err := r.Status().Update(ctx, Bookingsd); err != nil {
				log.Error(err, "Failed to update Bookingsd status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		log.Info("Creating a new Deployment",
			"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err = r.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create new Deployment",
				"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}

		// Deployment created successfully
		// We will requeue the reconciliation so that we can ensure the state
		// and move forward for the next operations
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		// Let's return the error for the reconciliation be re-trigged again
		return ctrl.Result{}, err
	}

	// The CRD API is defining that the Bookingsd type, have a BookingsdSpec.Size field
	// to set the quantity of Deployment instances is the desired state on the cluster.
	// Therefore, the following code will ensure the Deployment size is the same as defined
	// via the Size spec of the Custom Resource which we are reconciling.
	size := Bookingsd.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		if err = r.Update(ctx, found); err != nil {
			log.Error(err, "Failed to update Deployment",
				"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)

			// Re-fetch the Bookingsd Custom Resource before update the status
			// so that we have the latest state of the resource on the cluster and we will avoid
			// raise the issue "the object has been modified, please apply
			// your changes to the latest version and try again" which would re-trigger the reconciliation
			if err := r.Get(ctx, req.NamespacedName, Bookingsd); err != nil {
				log.Error(err, "Failed to re-fetch Bookingsd")
				return ctrl.Result{}, err
			}

			// The following implementation will update the status
			meta.SetStatusCondition(&Bookingsd.Status.Conditions, metav1.Condition{Type: typeAvailableBookingsd,
				Status: metav1.ConditionFalse, Reason: "Resizing",
				Message: fmt.Sprintf("Failed to update the size for the custom resource (%s): (%s)", Bookingsd.Name, err)})

			if err := r.Status().Update(ctx, Bookingsd); err != nil {
				log.Error(err, "Failed to update Bookingsd status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		// Now, that we update the size we want to requeue the reconciliation
		// so that we can ensure that we have the latest state of the resource before
		// update. Also, it will help ensure the desired state on the cluster
		return ctrl.Result{Requeue: true}, nil
	}

	// The following implementation will update the status
	meta.SetStatusCondition(&Bookingsd.Status.Conditions, metav1.Condition{Type: typeAvailableBookingsd,
		Status: metav1.ConditionTrue, Reason: "Reconciling",
		Message: fmt.Sprintf("Deployment for custom resource (%s) with %d replicas created successfully", Bookingsd.Name, size)})

	if err := r.Status().Update(ctx, Bookingsd); err != nil {
		log.Error(err, "Failed to update Bookingsd status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

// doFinalizerOperationsForBookingsd will perform the required operations before delete the CR.
func (r *BookingsdReconciler) doFinalizerOperationsForBookingsd(cr *bookingsv1alpha1.Bookingsd) {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.

	// Note: It is not recommended to use finalizers with the purpose of delete resources which are
	// created and managed in the reconciliation. These ones, such as the Deployment created on this reconcile,
	// are defined as depended of the custom resource. See that we use the method ctrl.SetControllerReference.
	// to set the ownerRef which means that the Deployment will be deleted by the Kubernetes API.
	// More info: https://kubernetes.io/docs/tasks/administer-cluster/use-cascading-deletion/

	log := log.FromContext(context.Background())

	// The following implementation will raise an event
	//r.Recorder.Event(cr, "Warning", "Deleting",
	//	fmt.Sprintf("Custom Resource %s is being deleted from the namespace %s",
	//		cr.Name,
	//		cr.Namespace))

	ok := controllerutil.RemoveFinalizer(cr, bookingsdFinalizer)
	if ok {
		if err := r.Update(context.Background(), cr); err != nil {
			log.Error(err, "Failed to remove finalizer from Bookingsd")
			return
		}
	}

}

// labelsForBookingsd returns the labels for selecting the resources
// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
func labelsForBookingsd(name string) map[string]string {
	var imageTag string
	image, initImage, err := imageForBookingsd()
	if err == nil {
		imageTag = strings.Split(image, ":")[1]
	}
	return map[string]string{
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/name":       "bookings",
		"app.kubernetes.io/version":    imageTag,
		"app.kubernetes.io/init":       initImage,
		"app.kubernetes.io/part-of":    "Bookingsd-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

// imageForBookingsd gets the Operand image which is managed by this controller
// from the BOOKINGS_IMAGE environment variable defined in the config/manager/manager.yaml
func imageForBookingsd() (image string, initImage string, errorFound error) {
	var imageEnvVar = "BOOKINGS_IMAGE"
	var imageInitEnvVar = "BOOKINGS_INIT_IMAGE"
	image, found := os.LookupEnv(imageEnvVar)
	if !found {
		return "", "", fmt.Errorf("Unable to find %s environment variable with the image", imageEnvVar)
	}

	initImage, found = os.LookupEnv(imageInitEnvVar)
	if !found {
		return "", "", fmt.Errorf("Unable to find %s environment variable with the image", imageInitEnvVar)
	}
	return image, initImage, nil
}

// SetupWithManager sets up the controller with the Manager.
// Note that the Deployment will be also watched in order to ensure its
// desirable state on the cluster
func (r *BookingsdReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bookingsv1alpha1.Bookingsd{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
