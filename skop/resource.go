package skop

import (
	"encoding/json"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Resource interface {
	metav1.Object
}

func makeResource(resourceType reflect.Type, source interface{}) (Resource, error) {
	dest := reflect.New(resourceType).Interface().(Resource)
	data, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &dest); err != nil {
		return nil, err
	}
	return dest, nil
}
