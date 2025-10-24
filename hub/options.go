package hub

import "time"

////////////////////////////////Server///////////////////////////////////////

type HubOption func(o *Hub)

func WithAddress(addr string) HubOption {
	return func(s *Hub) {
		s.address = addr
	}
}

func WithTimeout(timeout time.Duration) HubOption {
	return func(s *Hub) {
		s.timeout = timeout
	}
}
