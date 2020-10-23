package reconcile

import (
	"k8s.io/apimachinery/pkg/api/errors"
)

func Absence(delete func() error) error {
	err := delete()
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}
