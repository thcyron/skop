package reconcile

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/ericchiang/k8s"
	appsv1 "github.com/ericchiang/k8s/apis/apps/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/golang/mock/gomock"

	"github.com/thcyron/skop/skop/mock"
)

func TestDaemonSetExisting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client = mock.NewClient(ctrl)
		ctx    = context.Background()

		daemonSet = &appsv1.DaemonSet{
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
			Spec: &appsv1.DaemonSetSpec{
				MinReadySeconds: k8s.Int32(10),
			},
		}
		existingDaemonSet = &appsv1.DaemonSet{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("test"),
				Namespace: k8s.String("skop"),
			},
			Spec: &appsv1.DaemonSetSpec{
				MinReadySeconds: k8s.Int32(5),
			},
		}
	)

	client.
		EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq("test"), gomock.Any()).
		Do(func(_ context.Context, _ string, res k8s.Resource) {
			*res.(*appsv1.DaemonSet) = *existingDaemonSet
		}).
		Return(nil)

	client.
		EXPECT().
		Update(gomock.Eq(ctx), gomock.Any()).
		Do(func(_ context.Context, res k8s.Resource) {
			d := res.(*appsv1.DaemonSet)
			if !reflect.DeepEqual(d.Spec, daemonSet.Spec) {
				t.Errorf("unexpected spec in updated daemonset: %v", d.Spec)
			}
			if !reflect.DeepEqual(d.Metadata.Labels, daemonSet.Metadata.Labels) {
				t.Errorf("unexpected labels in updated daemonset: %v", d.Metadata.Labels)
			}
			if !reflect.DeepEqual(d.Metadata.Annotations, daemonSet.Metadata.Annotations) {
				t.Errorf("unexpected annotations in updated daemonset: %v", d.Metadata.Annotations)
			}
		}).
		Return(nil)

	if err := DaemonSet(ctx, client, daemonSet); err != nil {
		t.Fatal(err)
	}
}

func TestDaemonSetNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client = mock.NewClient(ctrl)
		ctx    = context.Background()

		daemonSet = &appsv1.DaemonSet{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("test"),
				Namespace: k8s.String("skop"),
			},
			Spec: &appsv1.DaemonSetSpec{
				MinReadySeconds: k8s.Int32(10),
			},
		}
	)

	client.
		EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq("test"), gomock.Any()).
		Return(&k8s.APIError{Code: http.StatusNotFound})

	client.
		EXPECT().
		Create(gomock.Eq(ctx), gomock.Eq(daemonSet)).
		Return(nil)

	if err := DaemonSet(ctx, client, daemonSet); err != nil {
		t.Fatal(err)
	}
}

func TestDaemonSetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client = mock.NewClient(ctrl)
		ctx    = context.Background()

		daemonSet = &appsv1.DaemonSet{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("test"),
				Namespace: k8s.String("skop"),
			},
			Spec: &appsv1.DaemonSetSpec{
				MinReadySeconds: k8s.Int32(10),
			},
		}
	)

	client.
		EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq("test"), gomock.Any()).
		Return(&k8s.APIError{Code: http.StatusInternalServerError})

	if err := DaemonSet(ctx, client, daemonSet); err == nil {
		t.Fatal("error expected")
	}
}
