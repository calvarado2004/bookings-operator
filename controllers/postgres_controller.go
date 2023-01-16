package controllers

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const postgresFinalizer = "bookings.calvarado04.com/finalizer"

// Definitions to manage status conditions
const (
	// typeAvailablePostgres represents the status of the Deployment reconciliation
	typeAvailablePostgres = "Available"
	// typeDegradedPostgres represents the status used when the custom resource is deleted and the finalizer operations are must to occur.
	typeDegradedPostgres = "Degraded"
)

// PostgresReconciler reconciles a Postgres object
type PostgresReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}
