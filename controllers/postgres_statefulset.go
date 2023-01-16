package controllers

import (
	cachev1alpha1 "github.com/calvarado2004/bookings-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// statefulSetForPostgres returns a Postgres StatefulSet object
func (r *PostgresReconciler) statefulSetForPostgres(Postgres *cachev1alpha1.Postgres) (*appsv1.StatefulSet, error) {
	ls := labelsForPostgres(Postgres.Name)
	replicas := Postgres.Spec.Size

	// Get the Operand image
	image, err := imageForPostgres()
	if err != nil {
		return nil, err
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Postgres.Name,
			Namespace: Postgres.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					SchedulerName: "stork",
					Containers: []corev1.Container{
						{
							Image:           image,
							Name:            "PostgresContainer",
							ImagePullPolicy: corev1.PullAlways,
						},
					},
				},
			},
			Replicas: &replicas,
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "data",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					StorageClassName: &Postgres.Spec.StorageClassName,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse(Postgres.Spec.StorageSize),
						},
					},
				},
			},
			},
			ServiceName: "postgres-svc",
		},
	}

	// Set the ownerRef for the StatefulSet
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(Postgres, sts, r.Scheme); err != nil {
		return nil, err
	}
	return sts, nil
}

func (r *PostgresReconciler) serviceForPostgres(Postgres *cachev1alpha1.Postgres) (servicePostgres *corev1.Service, err error) {
	// Define the Service
	servicePostgres = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Postgres.Name + "-svc",
			Namespace: Postgres.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "postgres",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "postgres",
					Port:       Postgres.Spec.ContainerPort,
					TargetPort: intstr.FromInt(int(Postgres.Spec.ContainerPort)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	return servicePostgres, nil
}
