package reconcile

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ConfigMap(ctx context.Context, cs *kubernetes.Clientset, configMap *corev1.ConfigMap) error {
	client := cs.CoreV1().ConfigMaps(configMap.Namespace)
	existing, err := client.Get(ctx, configMap.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = client.Create(ctx, configMap, metav1.CreateOptions{})
			return err
		}
		return err
	}
	existing.Labels = configMap.Labels
	existing.Annotations = configMap.Annotations
	existing.Data = configMap.Data
	existing.BinaryData = configMap.BinaryData
	existing.Immutable = configMap.Immutable
	_, err = client.Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func ConfigMapAbsence(ctx context.Context, cs *kubernetes.Clientset, configMap *corev1.ConfigMap) error {
	return Absence(func() error {
		return cs.CoreV1().ConfigMaps(configMap.Namespace).Delete(ctx, configMap.Name, metav1.DeleteOptions{})
	})
}
