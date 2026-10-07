package camera

import (
	"log"
	"sync"
	"time"
)

// Manager gerencia múltiplas câmeras.
type Manager struct {
	mu       sync.RWMutex
	cameras  map[int]*Camera
	notifier chan []byte
}

// NewManager cria uma nova instância do gerenciador.
func NewManager(notifier chan []byte) *Manager {
	return &Manager{
		cameras:  make(map[int]*Camera),
		notifier: notifier,
	}
}

// StartDiscovery tenta abrir câmeras em vários índices.
func (m *Manager) StartDiscovery(maxDevices int) {
	for i := 0; i < maxDevices; i++ {
		go m.tryOpen(i)
	}
	
	// Polling agressivo de reconexão (Hotplug)
	go func() {
		for {
			time.Sleep(3 * time.Second)
			m.mu.RLock()
			for i := 0; i < maxDevices; i++ {
				if _, ok := m.cameras[i]; !ok {
					go m.tryOpen(i)
				}
			}
			m.mu.RUnlock()
		}
	}()
}

func (m *Manager) tryOpen(deviceID int) {
	cam, err := NewCamera(deviceID, m.notifier)
	if err != nil {
		return // Não conseguiu abrir (câmera não existe ou ocupada)
	}
	
	log.Printf("Câmera conectada no dispositivo %d", deviceID)
	
	m.mu.Lock()
	m.cameras[deviceID] = cam
	m.mu.Unlock()
	
	cam.StartCapture()
	
	// Se StartCapture retornar, a câmera foi desconectada
	m.mu.Lock()
	delete(m.cameras, deviceID)
	m.mu.Unlock()
	log.Printf("⚠️ Câmera desconectada do dispositivo %d. Tentando reconectar em breve...", deviceID)
}

// GetCamera retorna uma câmera pelo ID, se existir.
func (m *Manager) GetCamera(id int) *Camera {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cameras[id]
}

// ListActiveIDs retorna a lista de IDs de câmeras conectadas.
func (m *Manager) ListActiveIDs() []int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var ids []int
	for id := range m.cameras {
		ids = append(ids, id)
	}
	return ids
}
