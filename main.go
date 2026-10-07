package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"webcam-go/camera"
	"webcam-go/server"
)

func downloadModel() {
	if _, err := os.Stat("haarcascade_frontalface_default.xml"); os.IsNotExist(err) {
		log.Println("Baixando modelo Haar Cascade FrontalFace (1MB)...")
		resp, err := http.Get("https://raw.githubusercontent.com/opencv/opencv/master/data/haarcascades/haarcascade_frontalface_default.xml")
		if err != nil {
			log.Fatalf("Erro ao baixar modelo: %v", err)
		}
		defer resp.Body.Close()

		out, err := os.Create("haarcascade_frontalface_default.xml")
		if err != nil {
			log.Fatalf("Erro ao criar arquivo: %v", err)
		}
		defer out.Close()
		io.Copy(out, resp.Body)
		log.Println("Modelo baixado com sucesso!")
	}
}

func main() {
	log.Println("Iniciando Sistema de Câmeras USB...")

	// Baixa o modelo de IA se não existir
	downloadModel()

	// Inicia rotina de auto-cleanup do disco
	server.StartCleanupWorker()

	// Inicializa SSE
	sse := server.NewSSEBroker()

	// Inicializa o gerenciador de câmeras
	camManager := camera.NewManager(sse.Notifier)
	
	// Inicia a busca por câmeras (dispositivos de 0 a 3 por exemplo)
	camManager.StartDiscovery(4)

	// Configura as rotas do servidor
	srv := server.NewServer(camManager, sse)
	
	// Rotas estáticas
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	
	log.Println("Servidor rodando na porta 8081")
	if err := http.ListenAndServe(":8081", srv.Router()); err != nil {
		log.Fatalf("Erro no servidor: %v", err)
	}
}
