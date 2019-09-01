package skop

import (
	"sync"

	"github.com/ericchiang/k8s"
)

type store struct {
	mu                  sync.RWMutex
	skipGenerationCheck bool
	resources           map[string]k8s.Resource
}

func (s *store) Get(name string) k8s.Resource {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.resources == nil {
		return nil
	}
	return s.resources[name]
}

func (s *store) Add(res k8s.Resource) bool {
	var (
		name     = res.GetMetadata().GetName()
		isUpdate bool
	)
	s.mu.Lock()
	if s.resources == nil {
		s.resources = make(map[string]k8s.Resource)
	}
	if existingResource := s.resources[name]; existingResource != nil {
		s.resources[name] = res
		if s.skipGenerationCheck {
			isUpdate = true
		} else {
			oldGen := existingResource.GetMetadata().GetGeneration()
			newGen := res.GetMetadata().GetGeneration()
			isUpdate = newGen > oldGen
		}
	} else {
		s.resources[name] = res
		isUpdate = true
	}
	s.mu.Unlock()
	return isUpdate
}

func (s *store) Remove(res k8s.Resource) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.resources == nil {
		return
	}
	name := res.GetMetadata().GetName()
	delete(s.resources, name)
}

func (s *store) Clear() {
	s.mu.Lock()
	s.resources = nil
	s.mu.Unlock()
}

func (s *store) All() []k8s.Resource {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]k8s.Resource, 0, len(s.resources))
	for _, res := range s.resources {
		all = append(all, res)
	}
	return all
}
