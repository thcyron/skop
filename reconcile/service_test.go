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

func TestServiceExisting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client = mock.NewClient(ctrl)
		ctx    = context.Background()

		service = &corev1.Service{
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
			Spec: &corev1.ServiceSpec{
				Selector: map[string]string{"foo": "bar"},
			},
		}
		existingService = &corev1.Service{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("test"),
				Namespace: k8s.String("skop"),
			},
			Spec: &corev1.ServiceSpec{
				ClusterIP: k8s.String("1.2.3.4"),
				Selector:  map[string]string{"foo": "bar"},
			},
		}
	)

	client.
		EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq("test"), gomock.Any()).
		Do(func(_ context.Context, _ string, res k8s.Resource) {
			*res.(*corev1.Service) = *existingService
		}).
		Return(nil)

	client.
		EXPECT().
		Update(gomock.Eq(ctx), gomock.Any()).
		Do(func(_ context.Context, res k8s.Resource) {
			s := res.(*corev1.Service)
			if s.Spec.ClusterIP == nil || *s.Spec.ClusterIP != "1.2.3.4" {
				t.Error("ClusterIP not carried over")
			}
			if !reflect.DeepEqual(s.Spec, service.Spec) {
				t.Errorf("unexpected spec in updated service: %v", s.Spec)
			}
			if !reflect.DeepEqual(s.Metadata.Labels, service.Metadata.Labels) {
				t.Errorf("unexpected labels in updated service: %v", s.Metadata.Labels)
			}
			if !reflect.DeepEqual(s.Metadata.Annotations, service.Metadata.Annotations) {
				t.Errorf("unexpected annotations in updated service: %v", s.Metadata.Annotations)
			}
		}).
		Return(nil)

	if err := Service(ctx, client, service); err != nil {
		t.Fatal(err)
	}
}

func TestServiceNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client = mock.NewClient(ctrl)
		ctx    = context.Background()

		service = &corev1.Service{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("test"),
				Namespace: k8s.String("skop"),
			},
			Spec: &corev1.ServiceSpec{
				Selector: map[string]string{"foo": "bar"},
			},
		}
	)

	client.
		EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq("test"), gomock.Any()).
		Return(&k8s.APIError{Code: http.StatusNotFound})

	client.
		EXPECT().
		Create(gomock.Eq(ctx), gomock.Eq(service)).
		Return(nil)

	if err := Service(ctx, client, service); err != nil {
		t.Fatal(err)
	}
}

func TestServiceGetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client = mock.NewClient(ctrl)
		ctx    = context.Background()

		service = &corev1.Service{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("test"),
				Namespace: k8s.String("skop"),
			},
			Spec: &corev1.ServiceSpec{
				Selector: map[string]string{"foo": "bar"},
			},
		}
	)

	client.
		EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq("test"), gomock.Any()).
		Return(&k8s.APIError{Code: http.StatusInternalServerError})

	if err := Service(ctx, client, service); err == nil {
		t.Fatal("error expected")
	}
}

func TestServiceCreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client = mock.NewClient(ctrl)
		ctx    = context.Background()

		service = &corev1.Service{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("test"),
				Namespace: k8s.String("skop"),
			},
			Spec: &corev1.ServiceSpec{
				Selector: map[string]string{"foo": "bar"},
			},
		}
	)

	client.
		EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq("test"), gomock.Any()).
		Return(&k8s.APIError{Code: http.StatusNotFound})

	client.
		EXPECT().
		Create(gomock.Eq(ctx), gomock.Eq(service)).
		Return(&k8s.APIError{Code: http.StatusInternalServerError})

	if err := Service(ctx, client, service); err == nil {
		t.Fatal("error expected")
	}
}
