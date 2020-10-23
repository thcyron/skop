package reconcile

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func PersistentVolumeClaim(ctx context.Context, cs *kubernetes.Clientset, pvc *corev1.PersistentVolumeClaim) error {
	client := cs.CoreV1().PersistentVolumeClaims(pvc.Namespace)
	existing, err := client.Get(ctx, pvc.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, pvc, metav1.CreateOptions{})
			return err
		}
		return err
	}
	existing.Labels = pvc.Labels
	existing.Annotations = pvc.Annotations
	existing.Spec.Resources = pvc.Spec.Resources
	_, err = client.Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func PersistentVolumeClaimAbsence(ctx context.Context, cs *kubernetes.Clientset, pvc *corev1.PersistentVolumeClaim) error {
	return Absence(func() error {
		return cs.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(ctx, pvc.Name, metav1.DeleteOptions{})
	})
}
