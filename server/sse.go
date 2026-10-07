package server

import (
	"fmt"
	"log"
	"net/http"
)

// SSEBroker gerencia os clientes conectados via Server-Sent Events
type SSEBroker struct {
	Notifier       chan []byte
	newClients     chan chan []byte
	closingClients chan chan []byte
	clients        map[chan []byte]bool
}

// NewSSEBroker inicializa e roda o loop principal do Broker
func NewSSEBroker() *SSEBroker {
	broker := &SSEBroker{
		Notifier:       make(chan []byte, 1),
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]bool),
	}

	go broker.listen()
	return broker
}

func (broker *SSEBroker) listen() {
	for {
		select {
		case s := <-broker.newClients:
			broker.clients[s] = true
			log.Printf("SSE Cliente conectado. Total: %d", len(broker.clients))
		case s := <-broker.closingClients:
			delete(broker.clients, s)
			log.Printf("SSE Cliente desconectado. Total: %d", len(broker.clients))
		case event := <-broker.Notifier:
			for clientMessageChan := range broker.clients {
				select {
				case clientMessageChan <- event:
				default:
					// Se o cliente não conseguir ler, não travamos o loop
				}
			}
		}
	}
}

func (broker *SSEBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming não suportado!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	messageChan := make(chan []byte)
	broker.newClients <- messageChan

	defer func() {
		broker.closingClients <- messageChan
	}()

	notify := r.Context().Done()
	go func() {
		<-notify
		broker.closingClients <- messageChan
	}()

	for {
		fmt.Fprintf(w, "event: MotionAlert\n")
		fmt.Fprintf(w, "data: %s\n\n", <-messageChan)
		flusher.Flush()
	}
}
