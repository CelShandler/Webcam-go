# Tasks: DVR Inteligente - Fase 3

- `[x]` Fase 3: Visão Computacional Avançada (IA e ROI)
  - `[x]` **1. IA / Human Detection (Haar Cascades)**:
    - `[x]` Criar utilitário no Go para baixar o `haarcascade_upperbody.xml` (se não existir localmente).
    - `[x]` Atualizar a lógica do `StartMotionDetection` para instanciar o `gocv.NewCascadeClassifier()`.
    - `[x]` Aplicar o Cascade no frame do MOG2 para confirmar falsos positivos.
  - `[x]` **2. Zonas (Region of Interest - ROI)**:
    - `[x]` Adicionar `roiX`, `roiY`, `roiW`, `roiH` na struct `Camera`.
    - `[x]` Implementar método na câmera para salvar as proporções do ROI.
    - `[x]` Cortar (`Region`) a Matriz do OpenCV nas proporções informadas antes de passar ao MOG2.
  - `[x]` **3. Interface de Desenho (Frontend)**:
    - `[x]` Criar um novo endpoint `/api/cameras/roi` (POST).
    - `[x]` Atualizar o Modal em `index.html` para ter um modo de "Edição de Zona".
    - `[x]` Criar a lógica JS de desenhar retângulo translúcido em cima do canvas/imagem.
    - `[x]` Enviar POST via HTMX/Fetch com as coordenadas percentuais.
