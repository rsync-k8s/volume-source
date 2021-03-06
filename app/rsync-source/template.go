package main

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	internalv1 "github.com/k8s-volume-copy/types/apis/demo.io/v1"
	"github.com/k8s-volume-copy/types/constant"
)

type templateConfig struct {
	name      string
	namespace string
	rsync     internalv1.RsyncSourceSpec
}

func templateConfigFromRsyncSource(cr internalv1.RsyncSource) (*templateConfig, error) {
	tc := &templateConfig{
		name:      cr.GetName(),
		namespace: cr.GetNamespace(),
		rsync:     cr.Spec,
	}
	return tc, nil
}

func (tc *templateConfig) getDeploymentTemplate() *appsv1.Deployment {
	nodeSelector := make(map[string]string)
	if tc.rsync.HostName != "" {
		nodeSelector[constant.K8SIOHostName] = tc.rsync.HostName
	}

	deploy := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: tc.name,
			Labels: map[string]string{
				constant.CreatedByLabel: constant.ComponentNameRsyncSourceController,
				constant.NameLabel:      tc.name,
				constant.AppLabel:       tc.name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: tc.rsync.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					constant.CreatedByLabel: constant.ComponentNameRsyncSourceController,
					constant.NameLabel:      tc.name,
					constant.AppLabel:       tc.name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						constant.CreatedByLabel: constant.ComponentNameRsyncSourceController,
						constant.NameLabel:      tc.name,
						constant.AppLabel:       tc.name,
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector: nodeSelector,
					Containers: []corev1.Container{
						{
							Name:            "rsync-daemon",
							Image:           tc.rsync.Image,
							ImagePullPolicy: corev1.PullAlways,
							Env: []corev1.EnvVar{
								{
									Name:  "RSYNC_PASSWORD",
									Value: tc.rsync.Password,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "rsync-daemon",
									ContainerPort: 873,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      tc.rsync.Volume.Name,
									MountPath: "/data",
									ReadOnly:  true,
									MountPropagation: func() *corev1.MountPropagationMode {
										name := corev1.MountPropagationHostToContainer
										return &name
									}(),
								},
								{
									Name:      "config",
									MountPath: "/etc/rsyncd.con",
									SubPath:   "rsyncd.con",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						tc.rsync.Volume,
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: tc.name,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return &deploy
}

func (tc *templateConfig) getCmTemplate() *corev1.ConfigMap {
	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: tc.name,
			Labels: map[string]string{
				constant.CreatedByLabel: constant.ComponentNameRsyncSourceController,
				constant.NameLabel:      tc.name,
			},
		},
		Data: map[string]string{
			"rsyncd.conf": rsyncdconfig,
		},
	}
	return &cm
}

func (tc *templateConfig) getSvcTemplate() *corev1.Service {
	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: tc.name,
			Labels: map[string]string{
				constant.CreatedByLabel: constant.ComponentNameRsyncSourceController,
				constant.NameLabel:      tc.name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "rsync-daemon",
					Port:     873,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				constant.CreatedByLabel: constant.ComponentNameRsyncSourceController,
				constant.NameLabel:      tc.name,
				constant.AppLabel:       tc.name,
			},
		},
	}
	return &svc
}

var rsyncdconfig = `
# /etc/rsyncd.conf
# Minimal configuration file for rsync daemon
# See rsync(1) and rsyncd.conf(5) man pages for help
# This line is required by the /etc/init.d/rsyncd script
pid file = /var/run/rsyncd.pid
uid = 0
gid = 0
use chroot = yes
reverse lookup = no
[data]
    hosts deny = *
    hosts allow = 0.0.0.0/0
    read only = false
    path = /data
    auth users = , user:rw
    secrets file = /etc/rsyncd.secrets
    timeout = 600
    transfer logging = true
`
