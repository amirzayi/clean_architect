package bus

import "errors"

type inMemory struct {
	data map[string]chan []byte
}

func NewInMemoryDriver(queues []string) Driver {
	m := make(map[string]chan []byte)
	for _, queue := range queues {
		m[queue] = make(chan []byte)
	}
	return inMemory{data: m}
}

func (m inMemory) Publish(subject string, content []byte) error {
	ch, ok := m.data[subject]
	if !ok {
		return errors.New("queue doesn't exists")
	}
	go func() {
		ch <- content
	}()
	return nil
}

func (m inMemory) Subscribe(subject string) (<-chan []byte, <-chan error, error) {
	ch, ok := m.data[subject]
	if !ok {
		return nil, nil, errors.New("queue doesn't exists")
	}

	errCh := make(chan error)
	close(errCh)
	return ch, errCh, nil
}

// fix: call to close channels
func (m inMemory) Close() error {
	for _, ch := range m.data {
		close(ch)
	}
	return nil
}
