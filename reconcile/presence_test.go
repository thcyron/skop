package reconcile

import (
	"context"
	"net/http"
	"testing"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	"github.com/golang/mock/gomock"

	"github.com/thcyron/skop/skop/mock"
)

func TestPresence(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client    = mock.NewClient(ctrl)
		ctx       = context.Background()
		configMap = &corev1.ConfigMap{}
	)

	client.
		EXPECT().
		Create(gomock.Eq(ctx), gomock.Eq(configMap)).
		Return(nil)

	if err := Presence(ctx, client, configMap); err != nil {
		t.Fatal(err)
	}
}

func TestPresenceConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client    = mock.NewClient(ctrl)
		ctx       = context.Background()
		configMap = &corev1.ConfigMap{}
	)

	client.
		EXPECT().
		Create(gomock.Eq(ctx), gomock.Eq(configMap)).
		Return(&k8s.APIError{Code: http.StatusConflict})

	if err := Presence(ctx, client, configMap); err != nil {
		t.Fatal(err)
	}
}

func TestPresenceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		client    = mock.NewClient(ctrl)
		ctx       = context.Background()
		configMap = &corev1.ConfigMap{}
	)

	client.
		EXPECT().
		Create(gomock.Eq(ctx), gomock.Eq(configMap)).
		Return(&k8s.APIError{Code: http.StatusInternalServerError})

	if err := Presence(ctx, client, configMap); err == nil {
		t.Fatal("error expected")
	}
}
