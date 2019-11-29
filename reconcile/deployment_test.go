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

func TestDeploymentExisting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client = mock.NewClient(ctrl)
		ctx    = context.Background()

		deployment = &appsv1.Deployment{
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
			Spec: &appsv1.DeploymentSpec{
				Replicas: k8s.Int32(2),
			},
		}
		existingDeployment = &appsv1.Deployment{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("test"),
				Namespace: k8s.String("skop"),
			},
			Spec: &appsv1.DeploymentSpec{
				Replicas: k8s.Int32(1),
			},
		}
	)

	client.
		EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq("test"), gomock.Any()).
		Do(func(_ context.Context, _ string, res k8s.Resource) {
			*res.(*appsv1.Deployment) = *existingDeployment
		}).
		Return(nil)

	client.
		EXPECT().
		Update(gomock.Eq(ctx), gomock.Any()).
		Do(func(_ context.Context, res k8s.Resource) {
			d := res.(*appsv1.Deployment)
			if !reflect.DeepEqual(d.Spec, deployment.Spec) {
				t.Errorf("unexpected spec in updated deployment: %v", d.Spec)
			}
			if !reflect.DeepEqual(d.Metadata.Labels, deployment.Metadata.Labels) {
				t.Errorf("unexpected labels in updated deployment: %v", d.Metadata.Labels)
			}
			if !reflect.DeepEqual(d.Metadata.Annotations, deployment.Metadata.Annotations) {
				t.Errorf("unexpected annotations in updated deployment: %v", d.Metadata.Annotations)
			}
		}).
		Return(nil)

	if err := Deployment(ctx, client, deployment); err != nil {
		t.Fatal(err)
	}
}

func TestDeploymentNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client = mock.NewClient(ctrl)
		ctx    = context.Background()

		deployment = &appsv1.Deployment{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("test"),
				Namespace: k8s.String("skop"),
			},
			Spec: &appsv1.DeploymentSpec{
				Replicas: k8s.Int32(2),
			},
		}
	)

	client.
		EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq("test"), gomock.Any()).
		Return(&k8s.APIError{Code: http.StatusNotFound})

	client.
		EXPECT().
		Create(gomock.Eq(ctx), gomock.Eq(deployment)).
		Return(nil)

	if err := Deployment(ctx, client, deployment); err != nil {
		t.Fatal(err)
	}
}

func TestDeploymentError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client = mock.NewClient(ctrl)
		ctx    = context.Background()

		deployment = &appsv1.Deployment{
			Metadata: &metav1.ObjectMeta{
				Name:      k8s.String("test"),
				Namespace: k8s.String("skop"),
			},
			Spec: &appsv1.DeploymentSpec{
				Replicas: k8s.Int32(2),
			},
		}
	)

	client.
		EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq("test"), gomock.Any()).
		Return(&k8s.APIError{Code: http.StatusInternalServerError})

	if err := Deployment(ctx, client, deployment); err == nil {
		t.Fatal("error expected")
	}
}
