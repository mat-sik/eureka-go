package name

import (
	"sync"
)

type Store struct {
	nameToHostStatuses map[string]map[string]Status
	lock               *sync.RWMutex
}

func (s *Store) addNew(name string, host string) {
	s.Put(name, host, Unknown)
}

func (s *Store) Put(name string, host string, status Status) {
	s.lock.Lock()
	defer s.lock.Unlock()

	hostStatuses, ok := s.nameToHostStatuses[name]
	if !ok {
		hostStatuses = make(map[string]Status)
		s.nameToHostStatuses[name] = hostStatuses
	}

	hostStatuses[host] = status
}

func (s *Store) Remove(name string, host string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	ips, ok := s.nameToHostStatuses[name]
	if !ok {
		return false
	}

	if _, ok = ips[host]; ok {
		delete(ips, host)

		if len(ips) == 0 {
			delete(s.nameToHostStatuses, name)
		}

		return true
	}
	return false
}

func (s *Store) Get(name string) []HostStatus {
	s.lock.RLock()
	defer s.lock.RUnlock()

	hostStatuses, _ := s.nameToHostStatuses[name]

	result := make([]HostStatus, 0, len(hostStatuses))
	for ipString := range hostStatuses {
		result = append(result, HostStatus{
			IP:     ipString,
			Status: hostStatuses[ipString],
		})
	}

	return result
}

func NewStore() Store {
	return Store{
		nameToHostStatuses: make(map[string]map[string]Status),
		lock:               &sync.RWMutex{},
	}
}
