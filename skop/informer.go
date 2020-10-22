package skop

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type informer interface {
	Get(key string) Resource
	Keys() []string
	Key(Resource) string
	Run(stopCh <-chan struct{}, update func(Resource))
}

type k8sInformer struct {
	resourceType reflect.Type
	informer     cache.SharedIndexInformer
	store        cache.Store
}

func newK8sInformer(
	config *rest.Config,
	namespace string,
	gvr schema.GroupVersionResource,
	resourceType reflect.Type,
) (*k8sInformer, error) {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, 0, namespace, nil)
	informer := factory.ForResource(gvr).Informer()

	return &k8sInformer{
		resourceType: resourceType,
		informer:     informer,
		store:        informer.GetStore(),
	}, nil
}

func (i *k8sInformer) Run(stopCh <-chan struct{}, update func(Resource)) {
	i.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			res, err := makeResource(i.resourceType, obj)
			if err != nil {
				panic(err)
			}
			update(res)
		},
		UpdateFunc: func(oldObj, obj interface{}) {
			res, err := makeResource(i.resourceType, obj)
			if err != nil {
				panic(err)
			}
			update(res)
		},
	})
	i.informer.Run(stopCh)
}

func (i *k8sInformer) Get(key string) Resource {
	obj, exists, err := i.store.GetByKey(key)
	if err != nil {
		panic(err)
	}
	if !exists {
		return nil
	}
	res, err := makeResource(i.resourceType, obj)
	if err != nil {
		panic(err)
	}
	return res
}

func (i *k8sInformer) Keys() []string {
	return i.store.ListKeys()
}

func (i *k8sInformer) Key(res Resource) string {
	key, _ := cache.MetaNamespaceKeyFunc(res)
	return key
}
