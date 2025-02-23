package name

import (
	"net"
	"sync"
)

type Store struct {
	nameToIp map[string]map[string]struct{}
	lock     *sync.RWMutex
}

func (s *Store) Add(name string, ip net.IP) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ips, ok := s.nameToIp[name]
	if !ok {
		ips = make(map[string]struct{})
		s.nameToIp[name] = ips
	}

	ips[ip.String()] = struct{}{}
}

func (s *Store) Remove(name string, ip net.IP) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	ips, ok := s.nameToIp[name]
	if !ok {
		return false
	}

	stringIP := ip.String()
	if _, ok = ips[stringIP]; ok {
		delete(ips, stringIP)

		if len(ips) == 0 {
			delete(s.nameToIp, name)
		}

		return true
	}
	return false
}

func (s *Store) Get(name string) []net.IP {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ips, _ := s.nameToIp[name]

	result := make([]net.IP, 0, len(ips))
	for ip := range ips {
		result = append(result, net.ParseIP(ip))
	}

	return result
}

func NewStore() Store {
	return Store{
		nameToIp: make(map[string]map[string]struct{}),
		lock:     &sync.RWMutex{},
	}
}
