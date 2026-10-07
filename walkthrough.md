# Walkthrough: Fase 3 do DVR Inteligente

A **Fase 3** foi um divisor de águas para o nosso projeto. Transformamos a detecção de movimento simples em um sistema inteligente blindado contra alarmes falsos, empregando Inteligência Artificial (Filtro de Humanos) e Zonas de ROI (Region of Interest) customizáveis visualmente.

## O Que Foi Implementado

### 1. Download Automático da Inteligência Artificial
- No momento em que você subir o servidor (`go run .`), o sistema verificará se o modelo base de IA existe.
- Se não existir, ele vai conectar silenciosamente aos servidores oficiais do OpenCV e fará o download do arquivo `haarcascade_frontalface_default.xml` (~1 MB) diretamente para a pasta do seu projeto. Escolhemos esse modelo por ser extremamente rápido na CPU.

### 2. Validador de Movimento (MOG2 + Cascade Classifier)
A arquitetura do backend em `camera.go` agora possui um sistema de detecção em duas etapas, operando a 5 frames por segundo (para economizar CPU):
- **Passo 1 (Trigger):** O algoritmo antigo (Background Subtractor MOG2) escaneia os pixels em busca de qualquer alteração na tela.
- **Passo 2 (Autenticação IA):** Se o passo 1 notar uma mudança (vento, sombra, gato), ao invés de alertar o usuário e disparar a gravação, ele converte o frame para Tons de Cinza (Grayscale) e joga para dentro da Rede Neural do Haar Cascade. Se a rede não encontrar feições humanas, ela descarta o movimento. Se encontrar, inicia a gravação e sobe o alerta no SSE!

### 3. Zonas de Interesse Desenháveis (ROI)
Implementamos uma ferramenta robusta no frontend (Modal) para você delimitar a área que quer monitorar:
- **Ferramenta de Edição:** No canto inferior esquerdo do modal maximizado de uma câmera, adicionamos o botão "Desenhar Zona".
- **Comportamento Interativo:** Ao clicar, o seu ponteiro muda para uma mira. Clique, segure e arraste sobre o vídeo ao vivo para pintar um retângulo vermelho (A Zona).
- **Submissão Geométrica:** Ao clicar em "Salvar Zona", um JavaScript faz a matemática avançada de normalização (convertendo o tamanho da tela para proporções de 0.0 a 1.0) e faz um POST silencioso (via Fetch API) para o novo endpoint `/api/cameras/roi`.
- **Corte a Laser:** O backend (`camera.go`) recebe essas proporções e passa a cortar fisicamente o frame do OpenCV (`img.Region`) isolando apenas o seu retângulo, de modo que toda a física do Passo 1 e Passo 2 aconteçam exclusivamente dentro daquela caixa delimitada.

## Como Testar

> [!CAUTION]
> Reinicie o servidor para injetar o downloader da IA no `main.go`. Se for a primeira vez rodando, olhe o terminal: você verá a mensagem "Baixando modelo Haar Cascade FrontalFace...".

1. **Testando o ROI**:
   - Abra o sistema, maximize a Câmera (clicando nela no grid).
   - Clique em **Desenhar Zona** e desenhe um quadrado numa área sem movimento.
   - Clique em **Salvar Zona**.
   - Feche o modal e balance as mãos no lado de fora da sua zona. Você notará que as caixas verdes do MOG2 no preview do grid deixarão de aparecer e nenhuma notificação surgirá.

2. **Testando a Inteligência Artificial**:
   - (Com ou Sem a zona de ROI ativa), passe a mão na frente da câmera escondendo seu rosto. O MOG2 verá a mão, mas a IA barrará a notificação.
   - Agora, coloque o seu rosto na frente da câmera. Após alguns instantes, o Alerta Vermelho saltará na tela confirmando "Humano Detectado" e gravando a evidência no disco!
