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

// deploymentForBookingsd returns a Bookingsd Deployment object
func (r *BookingsdReconciler) deploymentForBookingsd(Bookingsd *cachev1alpha1.Bookingsd) (*appsv1.Deployment, error) {
	ls := labelsForBookingsd(Bookingsd.Name)
	replicas := Bookingsd.Spec.Size

	LabelSelectorRequirementVar := metav1.LabelSelectorRequirement{
		Key:      "app.kubernetes.io/name",
		Operator: "In",
		Values:   []string{"bookings"},
	}

	PodAffinityTermVar := corev1.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				LabelSelectorRequirementVar,
			},
		},
		TopologyKey: "kubernetes.io/hostname",
	}

	AffinityVar := corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				PodAffinityTermVar,
			},
		},
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Bookingsd.Name,
			Namespace: Bookingsd.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					SchedulerName: "stork",
					Affinity:      &AffinityVar,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					InitContainers: []corev1.Container{{
						Image:           Bookingsd.Spec.InitContainerImage,
						Name:            "init-bookings",
						ImagePullPolicy: corev1.PullAlways,
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: &[]bool{true}[0],
						},
						Env: []corev1.EnvVar{
							{
								Name:  "DB_SERVER",
								Value: Bookingsd.Spec.DbServer,
							},
							{
								Name:  "DB_PORT",
								Value: Bookingsd.Spec.DbPort,
							},
							{
								Name:  "DB_USER",
								Value: Bookingsd.Spec.DbUser,
							},
							{
								Name:  "DB_NAME",
								Value: Bookingsd.Spec.DbName,
							},
							{
								Name: "DB_PASSWORD",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										Key: "postgresql-password",
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "postgres-secrets",
										},
									},
								},
							},
						},
					}},
					Containers: []corev1.Container{{
						Image:           Bookingsd.Spec.ContainerImage,
						Name:            "bookings",
						ImagePullPolicy: corev1.PullAlways,
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             &[]bool{true}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
						},
						Env: []corev1.EnvVar{
							{
								Name:  "MAILHOG_HOST",
								Value: Bookingsd.Spec.MailhogHost,
							},
							{
								Name:  "MAILHOG_PORT",
								Value: Bookingsd.Spec.MailhogPort,
							},
							{
								Name:  "DB_SERVER",
								Value: Bookingsd.Spec.DbServer,
							},
							{
								Name:  "DB_PORT",
								Value: Bookingsd.Spec.DbPort,
							},
							{
								Name:  "DB_USER",
								Value: Bookingsd.Spec.DbUser,
							},
							{
								Name:  "DB_NAME",
								Value: Bookingsd.Spec.DbName,
							},
							{
								Name: "DB_PASSWORD",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										Key: "postgresql-password",
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "postgres-secrets",
										},
									},
								},
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: Bookingsd.Spec.ContainerPort,
							Name:          "http",
							Protocol:      corev1.ProtocolTCP,
						}},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("10m"),
								corev1.ResourceMemory: resource.MustParse("10Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("100Mi"),
							},
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{
									Command: []string{"sh", "-ec", "wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/bookings|| exit 1"},
								},
							},
							InitialDelaySeconds: 7,
							TimeoutSeconds:      5,
							PeriodSeconds:       10,
							SuccessThreshold:    1,
							FailureThreshold:    6,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{
									Command: []string{"sh", "-ec", "wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/bookings|| exit 1"},
								},
							},
							InitialDelaySeconds: 7,
							TimeoutSeconds:      5,
							PeriodSeconds:       10,
							SuccessThreshold:    1,
							FailureThreshold:    6,
						},
					}},
				},
			},
		},
	}

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(Bookingsd, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
}

func (r *BookingsdReconciler) serviceForBookings(Bookingsd *cachev1alpha1.Bookingsd) (serviceBookings *corev1.Service, err error) {
	// Define the Service
	serviceBookings = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Bookingsd.Name + "-svc",
			Namespace: Bookingsd.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app.kubernetes.io/name": "bookings",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       Bookingsd.Spec.ContainerPort,
					TargetPort: intstr.FromInt(int(Bookingsd.Spec.ContainerPort)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	return serviceBookings, nil
}
