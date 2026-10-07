# DVR Inteligente (Sistema de Câmeras USB com Go e HTMX)

Este projeto é um sistema de monitoramento de câmeras USB (DVR Inteligente) desenvolvido com **Go**, **GoCV (OpenCV)** e **HTMX**. Ele permite a visualização e monitoramento simultâneo de múltiplas câmeras conectadas via USB, aplicando técnicas de Visão Computacional para detecção de movimento e pessoas.

## 🌟 Características e Capacidades

- **Múltiplas Câmeras Simultâneas**: Suporta leitura e transmissão concorrente de várias câmeras USB utilizando goroutines e mutexes para evitar condições de corrida.
- **Streaming MJPEG**: Transmissão contínua em tempo real (MJPEG) através de endpoints HTTP (ex: `/stream/{id}`), sem a necessidade de tecnologias pesadas como WebRTC.
- **Dashboard Responsivo (HTMX + CSS Grid)**: Interface leve e dinâmica alimentada por HTMX que atualiza o estado das câmeras em tempo real sem recarregar a página.
- **Detecção de Dispositivos**: Busca automática por câmeras USB disponíveis no sistema.
- **Visão Computacional e Inteligência Artificial**:
  - **Detecção de Movimento**: Identificação de alterações nos frames capturados.
  - **Detecção Humana (AI)**: Integração com modelo de inteligência artificial Haar Cascades (`haarcascade_frontalface_default.xml` / `upperbody`) para detectar presença humana e reduzir falsos positivos. O modelo é baixado automaticamente pelo sistema se não estiver presente.
  - **Zonas de Interesse (ROI)**: Permite desenhar e definir na própria interface Web uma Região de Interesse (Region of Interest). A câmera passará a monitorar movimentos e pessoas apenas dentro da área demarcada.
- **Gerenciamento de Disco Automático**: Rotina de limpeza (auto-cleanup) em segundo plano para arquivos e capturas antigas, evitando lotar o armazenamento do servidor.
- **Eventos em Tempo Real (SSE)**: Integração via Server-Sent Events (SSE) para notificar o frontend sobre eventos como quedas, reconexões de câmeras, ou alertas de detecção.

## ⚙️ Pré-requisitos (Windows)

- [Go](https://golang.org/dl/) 1.25 ou superior.
- [CMake](https://cmake.org/download/). Pode ser instalado via winget: `winget install Kitware.CMake`
- [MinGW-w64](https://www.msys2.org/) (com `mingw32-make` disponível no PATH do sistema).

## 🚀 Como Compilar e Executar

### 1. Compilar o OpenCV para GoCV

O GoCV exige que o OpenCV esteja compilado no sistema. O projeto inclui um script auxiliar para automatizar esse processo complexo no Windows.

Abra o **PowerShell** como Administrador na raiz do projeto e execute o script:
```powershell
.\setup_windows.ps1
```
*Atenção: A compilação a partir do código-fonte pode demorar de 30 minutos a 1 hora.*

### 2. Configurar o PATH

Se a compilação for bem-sucedida, você deve adicionar o diretório de binários compilados ao **PATH** do seu sistema Windows.
Adicione o seguinte caminho:
```
C:\opencv\build\install\x64\mingw\bin
```
*(Reinicie o terminal após adicionar ao PATH)*

### 3. Executar o Servidor

Após baixar as dependências e configurar o OpenCV, basta executar o servidor através do comando padrão do Go na raiz do projeto:

```bash
go run .
```

Durante a inicialização, o servidor irá verificar a presença dos modelos de IA necessários (Haar Cascades) e fará o download caso não existam. 

### 4. Acessar o Dashboard

Com o servidor rodando, abra seu navegador de preferência e acesse:
**http://localhost:8081**

Pela interface, você poderá visualizar o grid de câmeras detectadas e configurar as Zonas de Interesse (ROI).

## 📦 Distribuição e Compilação (Build)

O projeto possui um script automatizado (`build_dist.ps1`) para gerar a versão final para distribuição. Ele compila o executável, junta as dependências do OpenCV, inclui as pastas de design (frontend) e compacta tudo em um arquivo `.zip` pronto para envio.

### Como gerar o pacote final (.zip):

1. **Certifique-se de que o servidor não está rodando** (caso esteja rodando no terminal, encerre pressionando `Ctrl + C`).
2. Abra o terminal do PowerShell na raiz do projeto e execute:
   ```powershell
   .\build_dist.ps1
   ```
3. O script irá gerar um arquivo chamado **`dvr_inteligente_dist.zip`** na pasta. Esse pacote já contém as DLLs exigidas e não precisa de instalação de bibliotecas adicionais por parte do cliente!

*(Para mais explicações manuais detalhadas, consulte o arquivo 👉 **[Compilacao.md](Compilacao.md)**)*
