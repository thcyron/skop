package reconcile

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/golang/mock/gomock"

	"github.com/thcyron/skop/skop/mock"
)

func TestConfigMapExisting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client = mock.NewClient(ctrl)
		ctx    = context.Background()

		configMap = &corev1.ConfigMap{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("test"),
				Namespace: k8s.String("skop"),
				Labels: map[string]string{
					"label": "label",
				},
				Annotations: map[string]string{
					"annotation": "annotation",
				},
			},
			Data:       map[string]string{"foo": "baz"},
			BinaryData: map[string][]byte{"foo": []byte("baz")},
		}
		existingConfigMap = &corev1.ConfigMap{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("test"),
				Namespace: k8s.String("skop"),
			},
			Data:       map[string]string{"foo": "bar"},
			BinaryData: map[string][]byte{"foo": []byte("bar")},
		}
	)

	client.
		EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq("test"), gomock.Any()).
		Do(func(_ context.Context, _ string, res k8s.Resource) {
			*res.(*corev1.ConfigMap) = *existingConfigMap
		}).
		Return(nil)

	client.
		EXPECT().
		Update(gomock.Eq(ctx), gomock.Any()).
		Do(func(_ context.Context, res k8s.Resource) {
			cm := res.(*corev1.ConfigMap)
			if !reflect.DeepEqual(cm.Data, configMap.Data) {
				t.Errorf("unexpected data in updated config map: %v", cm.Data)
			}
			if !reflect.DeepEqual(cm.BinaryData, configMap.BinaryData) {
				t.Errorf("unexpected binary data in updated config map: %v", cm.BinaryData)
			}
			if !reflect.DeepEqual(cm.Metadata.Labels, configMap.Metadata.Labels) {
				t.Errorf("unexpected labels in updated config map: %v", cm.Metadata.Labels)
			}
			if !reflect.DeepEqual(cm.Metadata.Annotations, configMap.Metadata.Annotations) {
				t.Errorf("unexpected annotations in updated config map: %v", cm.Metadata.Annotations)
			}
		}).
		Return(nil)

	if err := ConfigMap(ctx, client, configMap); err != nil {
		t.Fatal(err)
	}
}

func TestConfigMapNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client = mock.NewClient(ctrl)
		ctx    = context.Background()

		configMap = &corev1.ConfigMap{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("test"),
				Namespace: k8s.String("skop"),
			},
			Data:       map[string]string{"foo": "baz"},
			BinaryData: map[string][]byte{"foo": []byte("baz")},
		}
	)

	client.
		EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq("test"), gomock.Any()).
		Return(&k8s.APIError{Code: http.StatusNotFound})

	client.
		EXPECT().
		Create(gomock.Eq(ctx), gomock.Eq(configMap)).
		Return(nil)

	if err := ConfigMap(ctx, client, configMap); err != nil {
		t.Fatal(err)
	}
}

func TestConfigMapError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client = mock.NewClient(ctrl)
		ctx    = context.Background()

		configMap = &corev1.ConfigMap{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("test"),
				Namespace: k8s.String("skop"),
			},
			Data:       map[string]string{"foo": "baz"},
			BinaryData: map[string][]byte{"foo": []byte("baz")},
		}
	)

	client.
		EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq("test"), gomock.Any()).
		Return(&k8s.APIError{Code: http.StatusInternalServerError})

	if err := ConfigMap(ctx, client, configMap); err == nil {
		t.Fatal("error expected")
	}
}
