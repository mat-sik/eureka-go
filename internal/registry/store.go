package registry

import (
	"sync"
)

type Store struct {
	serviceIDToHostStatuses map[string]map[string]Status
	lock                    sync.RWMutex
}

func (s *Store) addNew(serviceID string, host string) {
	s.Put(serviceID, host, Unknown)
}

func (s *Store) Put(serviceID string, host string, status Status) {
	s.lock.Lock()
	defer s.lock.Unlock()

	hostStatuses, ok := s.serviceIDToHostStatuses[serviceID]
	if !ok {
		hostStatuses = make(map[string]Status)
		s.serviceIDToHostStatuses[serviceID] = hostStatuses
	}

	hostStatuses[host] = status
}

func (s *Store) Remove(serviceID string, host string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	ips, ok := s.serviceIDToHostStatuses[serviceID]
	if !ok {
		return false
	}

	if _, ok = ips[host]; ok {
		delete(ips, host)

		if len(ips) == 0 {
			delete(s.serviceIDToHostStatuses, serviceID)
		}

		return true
	}
	return false
}

func (s *Store) Get(serviceID string) []HostStatus {
	s.lock.RLock()
	defer s.lock.RUnlock()

	hostStatuses, _ := s.serviceIDToHostStatuses[serviceID]

	result := make([]HostStatus, 0, len(hostStatuses))
	for ipString := range hostStatuses {
		result = append(result, HostStatus{
			Host:   ipString,
			Status: hostStatuses[ipString],
		})
	}

	return result
}

func (s *Store) GetServiceIDsToHosts() map[string][]string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	result := make(map[string][]string, len(s.serviceIDToHostStatuses))
	for serviceID, hostStatuses := range s.serviceIDToHostStatuses {
		result[serviceID] = make([]string, 0, len(hostStatuses))
		for ipString := range hostStatuses {
			result[serviceID] = append(result[serviceID], ipString)
		}
	}

	return result
}

func NewStore() *Store {
	serviceIdToHostStatuses := make(map[string]map[string]Status)
	return NewStoreFrom(serviceIdToHostStatuses)
}

func NewStoreFrom(serviceIdToHostStatuses map[string]map[string]Status) *Store {
	return &Store{
		serviceIDToHostStatuses: serviceIdToHostStatuses,
		lock:                    sync.RWMutex{},
	}
}
