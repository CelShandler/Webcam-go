package camera

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"sync"
	"time"

	"gocv.io/x/gocv"
)

// Camera encapsula a leitura de um dispositivo de vídeo.
type Camera struct {
	deviceID int
	vc       *gocv.VideoCapture
	
	mu       sync.RWMutex
	frameBuf []byte // Armazena o frame atual em JPEG
	paused   bool   // Flag para pausar a captura
	
	notifier chan []byte // Canal SSE
	
	recordingMu sync.Mutex
	isRecording bool

	// Region of Interest (0.0 a 1.0)
	roiMu sync.RWMutex
	roiX, roiY, roiW, roiH float64
	
	// Estado atual da resolução (ex: "720p")
	res string
}

// NewCamera tenta inicializar uma câmera em um determinado índice.
func NewCamera(deviceID int, notifier chan []byte) (*Camera, error) {
	vc, err := gocv.VideoCaptureDevice(deviceID)
	if err != nil {
		return nil, fmt.Errorf("não foi possível abrir o dispositivo %d: %v", deviceID, err)
	}
	
	// Verifica se consegue ler o primeiro frame
	img := gocv.NewMat()
	defer img.Close()
	if ok := vc.Read(&img); !ok || img.Empty() {
		vc.Close()
		return nil, fmt.Errorf("câmera não retornou imagem no dispositivo %d", deviceID)
	}

	return &Camera{
		deviceID: deviceID,
		vc:       vc,
		notifier: notifier,
		roiW:     1.0, // Default: full width
		roiH:     1.0, // Default: full height
		res:      "720p", // Default string
	}, nil
}

// StartCapture inicia a leitura contínua dos frames da câmera.
func (c *Camera) StartCapture() {
	defer c.vc.Close()
	
	img := gocv.NewMat()
	defer img.Close()

	// Tentar ~30 FPS (1000ms / 30 ≈ 33ms)
	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.RLock()
		isPaused := c.paused
		c.mu.RUnlock()

		if isPaused {
			// Se estiver pausada, pula a leitura e envia o buffer antigo ou aguarda
			continue
		}

		if ok := c.vc.Read(&img); !ok || img.Empty() {
			// Pode significar que a câmera foi desconectada
			break
		}

		// Converter para std image
		stdImg, err := img.ToImage()
		if err != nil {
			continue
		}

		// Converter para JPEG na memória
		buf := new(bytes.Buffer)
		if err := jpeg.Encode(buf, stdImg, &jpeg.Options{Quality: 60}); err != nil {
			continue
		}

		// Atualizar o buffer thread-safe
		c.mu.Lock()
		c.frameBuf = buf.Bytes()
		c.mu.Unlock()
	}
}

// GetLatestFrameJpeg retorna a cópia do último frame lido.
func (c *Camera) GetLatestFrameJpeg() []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Retornamos uma cópia para evitar que modifiquem enquanto escrevem?
	// O slice copia o len/cap, o array subjacente é o mesmo.
	// Como substituímos o slice no StartCapture, ler o valor aqui é seguro.
	return c.frameBuf
}

// TogglePause inverte o estado de pausa da câmera
func (c *Camera) TogglePause() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = !c.paused
	return c.paused
}

// IsPaused retorna o status atual de pausa
func (c *Camera) IsPaused() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.paused
}

// StartRecording grava o feed de vídeo para um arquivo por um tempo determinado.
func (c *Camera) StartRecording(duration time.Duration, filename string) error {
	c.recordingMu.Lock()
	if c.isRecording {
		c.recordingMu.Unlock()
		return fmt.Errorf("já está gravando")
	}
	c.isRecording = true
	c.recordingMu.Unlock()

	defer func() {
		c.recordingMu.Lock()
		c.isRecording = false
		c.recordingMu.Unlock()
	}()

	// Pega um frame de exemplo para saber as dimensões
	img := gocv.NewMat()
	defer img.Close()
	if ok := c.vc.Read(&img); !ok || img.Empty() {
		return fmt.Errorf("falha ao ler da câmera para iniciar gravação")
	}

	fps := 20.0
	// Tentar MJPG como codec seguro
	writer, err := gocv.VideoWriterFile(filename, "MJPG", fps, img.Cols(), img.Rows(), true)
	if err != nil {
		return err
	}
	defer writer.Close()

	timeout := time.After(duration)
	ticker := time.NewTicker(time.Duration(1000/fps) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil
		case <-ticker.C:
			frame := gocv.NewMat()
			if ok := c.vc.Read(&frame); ok && !frame.Empty() {
				writer.Write(frame)
			}
			frame.Close()
		}
	}
}

// SetROI define a região de interesse para a câmera baseada em proporções (0.0 a 1.0).
func (c *Camera) SetROI(x, y, w, h float64) {
	c.roiMu.Lock()
	defer c.roiMu.Unlock()
	c.roiX = x
	c.roiY = y
	c.roiW = w
	c.roiH = h
}

// SetResolution altera as propriedades de largura e altura da captura USB
func (c *Camera) SetResolution(width, height float64, resStr string) {
	c.mu.Lock()
	c.res = resStr
	c.mu.Unlock()
	
	c.vc.Set(gocv.VideoCaptureFrameWidth, width)
	c.vc.Set(gocv.VideoCaptureFrameHeight, height)
	log.Printf("Resolução da Câmera %d alterada para %.0fx%.0f (%s)", c.deviceID, width, height, resStr)
}

// GetResolution retorna a string da resolução atual
func (c *Camera) GetResolution() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.res
}

// StartMotionDetection roda o Background Subtractor MOG2 e filtra com IA antes de gravar.
func (c *Camera) StartMotionDetection() {
	go func() {
		mog2 := gocv.NewBackgroundSubtractorMOG2()
		defer mog2.Close()

		cascade := gocv.NewCascadeClassifier()
		defer cascade.Close()
		
		if !cascade.Load("haarcascade_frontalface_default.xml") {
			log.Println("Aviso: Modelo Haar Cascade não encontrado! Verifique a raiz do projeto.")
		}

		img := gocv.NewMat()
		defer img.Close()

		fgMask := gocv.NewMat()
		defer fgMask.Close()

		gray := gocv.NewMat()
		defer gray.Close()

		ticker := time.NewTicker(200 * time.Millisecond) // 5 FPS para poupar CPU
		defer ticker.Stop()

		motionDuration := 0
		recording := false

		for range ticker.C {
			if c.IsPaused() {
				continue
			}

			if ok := c.vc.Read(&img); !ok || img.Empty() {
				continue
			}

			// Aplicar ROI
			c.roiMu.RLock()
			x := int(c.roiX * float64(img.Cols()))
			y := int(c.roiY * float64(img.Rows()))
			w := int(c.roiW * float64(img.Cols()))
			h := int(c.roiH * float64(img.Rows()))
			c.roiMu.RUnlock()

			rect := image.Rect(x, y, x+w, y+h)
			roiImg := img.Region(rect)

			// Aplicar MOG2 na região de interesse
			mog2.Apply(roiImg, &fgMask)
			
			// Calcular proporção de pixels brancos (movimento na ROI)
			whites := gocv.CountNonZero(fgMask)
			ratio := float64(whites) / float64(fgMask.Rows()*fgMask.Cols())

			hasMotion := ratio > 0.01 // 1% da zona ROI mudou

			if hasMotion {
				// Validação por IA (Filtro de Falsos Positivos)
				gocv.CvtColor(roiImg, &gray, gocv.ColorBGRToGray)
				rects := cascade.DetectMultiScale(gray)
				
				// Se a IA achar pelo menos 1 rosto/pessoa na zona em movimento
				if len(rects) > 0 {
					if !recording {
						recording = true
						motionDuration = 0
						
						// Inicia a gravação em background
						go func() {
							filename := fmt.Sprintf("capturas/motion_cam%d_%d.avi", c.deviceID, time.Now().Unix())
							c.StartRecording(10*time.Second, filename)
						}()
					}

					motionDuration++

					// Após ~10s de movimento confirmado pela IA
					if motionDuration >= 50 {
						if c.notifier != nil {
							alertID := fmt.Sprintf("alert-%d-%d", c.deviceID, time.Now().UnixNano())
							msg := fmt.Sprintf(`<div id="%s" class="bg-red-500/90 border border-red-400 text-white px-4 py-3 rounded relative mb-4 shadow-xl backdrop-blur-sm transition-opacity duration-1000" role="alert"><strong class="font-bold">Atenção!</strong><span class="block sm:inline"> Humano detectado na Câmera %d. O vídeo de 10s foi salvo. Continuar vigilância?</span><div class="mt-2 flex space-x-2"><button class="bg-white text-red-600 font-bold py-1 px-3 rounded hover:bg-slate-100 text-sm" onclick="this.closest('div[role=\'alert\']').remove()">Sim, ignorar</button></div><script>setTimeout(function(){ var el = document.getElementById('%s'); if(el) { el.classList.add('opacity-0'); setTimeout(function(){ el.remove(); }, 1000); } }, 10000);</script></div>`, alertID, c.deviceID, alertID)
							c.notifier <- []byte(msg)
						}
						motionDuration = 0
						recording = false
					}
				}
			} else {
				if motionDuration > 0 {
					motionDuration--
				}
				if motionDuration == 0 {
					recording = false
				}
			}
			
			roiImg.Close()
		}
	}()
}
