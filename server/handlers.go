package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"webcam-go/camera"
)

type Server struct {
	camManager *camera.Manager
	templates  *template.Template
	sseBroker  *SSEBroker
}

func NewServer(m *camera.Manager, sse *SSEBroker) *Server {
	// Carrega os templates HTML
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	return &Server{
		camManager: m,
		templates:  tmpl,
		sseBroker:  sse,
	}
}

func (s *Server) Router() *http.ServeMux {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/cameras/grid", s.handleGrid)
	mux.HandleFunc("/api/cameras/toggle", s.handleTogglePause)
	mux.HandleFunc("/api/cameras/photo", s.handlePhoto)
	mux.HandleFunc("/api/cameras/record", s.handleRecord)
	mux.HandleFunc("/api/cameras/motion", s.handleMotion)
	mux.HandleFunc("/api/cameras/roi", s.handleSetROI)
	mux.HandleFunc("/api/cameras/resolution", s.handleSetResolution)
	mux.HandleFunc("/api/gallery", s.handleGallery)
	mux.HandleFunc("/stream/", s.handleStream)
	
	// Rota SSE
	mux.Handle("/api/events", s.sseBroker)
	
	// Rota de Arquivos (Download/Visualização Direta)
	mux.Handle("/capturas/", http.StripPrefix("/capturas/", http.FileServer(http.Dir("capturas"))))
	
	return mux
}

// handleIndex renderiza a página inicial (base do HTMX)
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.templates.ExecuteTemplate(w, "index.html", nil)
}

// handleGrid retorna o HTML do grid com as câmeras ativas
func (s *Server) handleGrid(w http.ResponseWriter, r *http.Request) {
	ids := s.camManager.ListActiveIDs()
	
	type CamInfo struct {
		ID  int
		Res string
	}
	
	var infos []CamInfo
	for _, id := range ids {
		cam := s.camManager.GetCamera(id)
		if cam != nil {
			infos = append(infos, CamInfo{ID: id, Res: cam.GetResolution()})
		}
	}
	
	data := struct {
		Cameras []CamInfo
	}{
		Cameras: infos,
	}
	
	s.templates.ExecuteTemplate(w, "grid.html", data)
}

// handleStream envia o fluxo MJPEG de uma câmera específica
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	// Extrair o ID da URL, ex: /stream/0
	idStr := r.URL.Path[len("/stream/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	cam := s.camManager.GetCamera(id)
	if cam == nil {
		http.Error(w, "Câmera não encontrada", http.StatusNotFound)
		return
	}

	// Configurar headers para MJPEG
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	
	// Fechar a conexão caso o cliente se desconecte
	ctx := r.Context()
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
			frame := cam.GetLatestFrameJpeg()
			if len(frame) == 0 {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			// Escrever o header da parte
			fmt.Fprintf(w, "--frame\r\n")
			fmt.Fprintf(w, "Content-Type: image/jpeg\r\n")
			fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(frame))
			
			// Escrever o frame
			w.Write(frame)
			
			// Fim da parte
			fmt.Fprintf(w, "\r\n")
			
			// Um pequeno delay para não sobrecarregar a rede/cpu
			time.Sleep(33 * time.Millisecond) // ~30fps
		}
	}
}

// handleTogglePause pausa ou retoma o stream de uma câmera e retorna o novo status
func (s *Server) handleTogglePause(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	cam := s.camManager.GetCamera(id)
	if cam == nil {
		http.Error(w, "Câmera não encontrada", http.StatusNotFound)
		return
	}

	isPaused := cam.TogglePause()
	
	if isPaused {
		w.Write([]byte("Retomar"))
	} else {
		w.Write([]byte("Pausar"))
	}
}

// handlePhoto baixa o último frame da câmera como uma imagem JPEG
func (s *Server) handlePhoto(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	cam := s.camManager.GetCamera(id)
	if cam == nil {
		http.Error(w, "Câmera não encontrada", http.StatusNotFound)
		return
	}

	frame := cam.GetLatestFrameJpeg()
	if len(frame) == 0 {
		http.Error(w, "Nenhum frame disponível", http.StatusServiceUnavailable)
		return
	}

	// Salva na pasta capturas
	filename := fmt.Sprintf("capturas/foto_cam%d_%d.jpg", id, time.Now().Unix())
	if err := os.WriteFile(filename, frame, 0644); err != nil {
		log.Printf("Erro ao salvar foto no disco: %v", err)
	}

	// Forçar download como anexo
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"camera_%d_%d.jpg\"", id, time.Now().Unix()))
	w.Header().Set("Content-Length", strconv.Itoa(len(frame)))
	
	w.Write(frame)
}

// handleRecord inicia a gravação de um clipe de 5 segundos
func (s *Server) handleRecord(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	cam := s.camManager.GetCamera(id)
	if cam == nil {
		http.Error(w, "Câmera não encontrada", http.StatusNotFound)
		return
	}

	filename := fmt.Sprintf("capturas/manual_cam%d_%d.avi", id, time.Now().Unix())
	
	// Roda a gravação em background
	go func() {
		if err := cam.StartRecording(5*time.Second, filename); err != nil {
			log.Printf("Erro ao gravar: %v", err)
		}
	}()

	w.Write([]byte("Gravando..."))
}

// handleMotion ativa a vigilância por detecção de movimento
func (s *Server) handleMotion(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	cam := s.camManager.GetCamera(id)
	if cam == nil {
		http.Error(w, "Câmera não encontrada", http.StatusNotFound)
		return
	}

	// Ativa o sub-sistema MOG2 para a câmera
	cam.StartMotionDetection()

	w.Write([]byte("Vigiando Humanos na ROI"))
}

// handleSetROI recebe as coordenadas da região de interesse
func (s *Server) handleSetROI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método inválido", http.StatusMethodNotAllowed)
		return
	}
	
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	cam := s.camManager.GetCamera(id)
	if cam == nil {
		http.Error(w, "Câmera não encontrada", http.StatusNotFound)
		return
	}

	x, _ := strconv.ParseFloat(r.FormValue("x"), 64)
	y, _ := strconv.ParseFloat(r.FormValue("y"), 64)
	wid, _ := strconv.ParseFloat(r.FormValue("w"), 64)
	h, _ := strconv.ParseFloat(r.FormValue("h"), 64)

	cam.SetROI(x, y, wid, h)
	w.Write([]byte("Zona ROI salva!"))
}

// handleSetResolution ajusta a resolução de hardware da câmera
func (s *Server) handleSetResolution(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	res := r.FormValue("res")
	var width, height float64
	switch res {
	case "1080p":
		width, height = 1920, 1080
	case "720p":
		width, height = 1280, 720
	case "480p":
		width, height = 640, 480
	default:
		width, height = 1280, 720 // fallback 720p
	}

	cam := s.camManager.GetCamera(id)
	if cam == nil {
		http.Error(w, "Câmera não encontrada", http.StatusNotFound)
		return
	}

	// Pausa rapidamente, troca, e retoma
	wasPaused := cam.IsPaused()
	if !wasPaused {
		cam.TogglePause()
	}
	
	cam.SetResolution(width, height, res)
	
	if !wasPaused {
		cam.TogglePause()
	}

	w.Write([]byte(res)) // Retorna a string pra preencher alguma div de confirmação
}

// handleGallery lê os arquivos em /capturas e renderiza a sidebar
func (s *Server) handleGallery(w http.ResponseWriter, r *http.Request) {
	dir := "capturas"
	files, err := os.ReadDir(dir)
	
	type MediaItem struct {
		Name     string
		Path     string
		Size     string
		ModTime  string
		IsVideo  bool
	}
	
	var items []MediaItem
	if err == nil {
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			info, err := f.Info()
			if err != nil {
				continue
			}
			
			sizeMB := float64(info.Size()) / 1024.0 / 1024.0
			isVideo := filepath.Ext(f.Name()) == ".avi"
			
			items = append(items, MediaItem{
				Name:    f.Name(),
				Path:    "/capturas/" + f.Name(),
				Size:    fmt.Sprintf("%.1f MB", sizeMB),
				ModTime: info.ModTime().Format("02/01 15:04:05"),
				IsVideo: isVideo,
			})
		}
	}

	// Ordena mais novos primeiro
	sort.Slice(items, func(i, j int) bool {
		return items[i].ModTime > items[j].ModTime
	})

	s.templates.ExecuteTemplate(w, "gallery.html", items)
}

// ==========================================
// AUTO-CLEANUP
// ==========================================

// StartCleanupWorker inicia a verificação periódica de espaço do diretório capturas.
func StartCleanupWorker() {
	go func() {
		// Executa a limpeza a cada 1 hora
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		// Executa a primeira vez logo ao iniciar
		runCleanup()

		for range ticker.C {
			runCleanup()
		}
	}()
}

func runCleanup() {
	dir := "capturas"
	var limitBytes int64 = 2 * 1024 * 1024 * 1024 // 2 GB

	files, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("Erro ao ler diretório capturas: %v", err)
		return
	}

	var totalSize int64
	type fileInfo struct {
		path string
		info os.FileInfo
	}
	var fileList []fileInfo

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		totalSize += info.Size()
		fileList = append(fileList, fileInfo{path: filepath.Join(dir, f.Name()), info: info})
	}

	if totalSize <= limitBytes {
		return // Tudo tranquilo
	}

	log.Printf("Aviso: Pasta capturas com %d bytes (limite é 2 GB). Iniciando auto-cleanup...", totalSize)

	// Ordena do mais antigo para o mais novo
	sort.Slice(fileList, func(i, j int) bool {
		return fileList[i].info.ModTime().Before(fileList[j].info.ModTime())
	})

	for _, f := range fileList {
		if totalSize <= limitBytes {
			break
		}
		err := os.Remove(f.path)
		if err == nil {
			log.Printf("Auto-cleanup: removido arquivo antigo %s", f.path)
			totalSize -= f.info.Size()
		}
	}
}
