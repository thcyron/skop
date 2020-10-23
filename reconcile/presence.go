package reconcile

import (
	"k8s.io/apimachinery/pkg/api/errors"
)

func Presence(create, update func() error) error {
	err := update()
	if err == nil {
		return nil
	}
	if errors.IsNotFound(err) {
		err = create()
	}
	return err
}
