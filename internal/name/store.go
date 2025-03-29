package name

import (
	"net"
	"sync"
)

type Store struct {
	nameToHostStatuses map[string]map[string]Status
	lock               *sync.RWMutex
}

func (s *Store) addNew(name string, ip net.IP) {
	s.Add(name, ip, Unknown)
}

func (s *Store) Add(name string, ip net.IP, status Status) {
	s.lock.Lock()
	defer s.lock.Unlock()

	hostStatuses, ok := s.nameToHostStatuses[name]
	if !ok {
		hostStatuses = make(map[string]Status)
		s.nameToHostStatuses[name] = hostStatuses
	}

	hostStatuses[ip.String()] = status
}

func (s *Store) Remove(name string, ip net.IP) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	ips, ok := s.nameToHostStatuses[name]
	if !ok {
		return false
	}

	stringIP := ip.String()
	if _, ok = ips[stringIP]; ok {
		delete(ips, stringIP)

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
