# Implementation Plan: Fase 3 - Visão Computacional Avançada (IA e ROI)

Avançando no roteiro do nosso **DVR Inteligente**, entraremos na Fase 3, focada em tornar a detecção de movimento incrivelmente precisa usando Inteligência Artificial e Delimitação Espacial.

## User Review Required

> [!WARNING]
> Rodar Inteligência Artificial em cima dos frames das câmeras aumentará consideravelmente o uso de CPU/GPU do seu servidor, especialmente se você tiver 4 câmeras rodando simultaneamente.

## Open Questions

> [!CAUTION]
> 1. **Filtro de IA (Human Detection):** Modelos como o YOLO-tiny requerem o download prévio de arquivos "pesos" (`.weights` de ~25MB) para funcionarem, mas são precisos para achar o corpo inteiro. Modelos clássicos como o *Haar Cascade* (usados para achar Rosto/Tronco) são minúsculos (1MB) e super rápidos na CPU.
>    - Qual você prefere? Posso implementar o **Haar Cascade FrontalFace/UpperBody** que não pesa nada e é nativo do OpenCV, ou você prefere que eu inclua o download automatizado do **YOLO-tiny**?
> 2. **Desenhando a Zona (ROI):** Para a Região de Interesse, precisaremos desenhar um retângulo vermelho translúcido por cima do vídeo da câmera para definir a zona de alarme (ex: apenas em cima da maçaneta de uma porta). Tudo bem se eu adicionar uma nova ferramenta (um botão "Desenhar Zona") apenas na visão maximizada do modal?

## Proposed Changes

### 1. Zonas de Detecção (Region of Interest - ROI)
#### [MODIFY] `templates/index.html` (Modal JS)
- Adicionar evento de clique-e-arraste no Modal (apenas quando em modo de "Edição de Zona").
- Esse evento criará uma `div` translúcida flutuante delimitando um retângulo (X, Y, Largura, Altura).
- Ao soltar o mouse, o HTMX ou Fetch enviará um POST para a API com essas coordenadas (normalizadas de 0 a 1 em relação à largura/altura do vídeo).

#### [MODIFY] `camera/camera.go`
- A struct `Camera` receberá novos campos `roiX, roiY, roiW, roiH float64`.
- A *goroutine* de `StartMotionDetection()` (MOG2) fará um "recorte" (`img.Region()`) do frame principal com base nessas proporções antes de passar o frame pro MOG2, ignorando o resto do mundo!

### 2. Filtro de Detecção Humana (Falsos Positivos)
#### [MODIFY] `camera/camera.go`
- O detector MOG2 passará a ser apenas um gatilho (Trigger).
- Quando o MOG2 detectar que houve "alterações nos pixels" (uma folha balançando, cachorro passando, ou pessoa), em vez de gravar direto, ele enviará aquele frame para a IA (CascadeClassifier ou YOLO).
- Se a IA validar que aqueles pixels formam um corpo humano (retornar bounding boxes da detecção), aí sim o flag de "Gravar" e a contagem dos 10 segundos são acionados.

#### [NEW] `server/download_models.go` (Caso escolhamos YOLO)
- Se a escolha for YOLO, criaremos uma função rodando no `main.go` que faz um `http.Get` para os servidores oficias para baixar os arquivos `.weights` no primeiro uso, salvando na pasta `/models/`.

## Verification Plan

### Automated Tests
- Testar recorte geométrico da Matriz (Mat) do OpenCV para o ROI.

### Manual Verification
1. Abrir modal, desenhar zona ROI no canto superior esquerdo e balançar a mão no canto direito. O alarme não deve tocar.
2. Ativar detecção de humanos, jogar uma bolinha (ou objeto) na frente da tela. O MOG2 vai ver a mudança de pixels, mas a IA vai barrar a notificação.
3. Colocar o rosto na câmera. A IA deve liberar e disparar a notificação e gravação no servidor.
