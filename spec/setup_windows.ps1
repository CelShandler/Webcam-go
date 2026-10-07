# setup_windows.ps1
# Script para facilitar a compilação do OpenCV para o GoCV no Windows

Write-Host "=======================================================" -ForegroundColor Cyan
Write-Host "   Configuração de Dependências - GoCV / OpenCV        " -ForegroundColor Cyan
Write-Host "=======================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "AVISO: Este processo irá compilar o OpenCV a partir do código-fonte." -ForegroundColor Yellow
Write-Host "Pode demorar de 30 minutos a mais de 1 hora dependendo do seu computador." -ForegroundColor Yellow
Write-Host ""

# Adiciona caminhos padrão ao PATH temporário da sessão para evitar ter que reiniciar o terminal
$env:PATH = "C:\Program Files\CMake\bin;C:\msys64\mingw64\bin;C:\msys64\usr\bin;" + $env:PATH

# Verifica dependências básicas
$missing = $false

if (-not (Get-Command cmake -ErrorAction SilentlyContinue)) {
    Write-Host "[X] CMake não encontrado!" -ForegroundColor Red
    Write-Host "    Execute no PowerShell como Admin: winget install Kitware.CMake" -ForegroundColor Yellow
    $missing = $true
} else {
    Write-Host "[OK] CMake encontrado." -ForegroundColor Green
}

if (-not (Get-Command mingw32-make -ErrorAction SilentlyContinue)) {
    Write-Host "[X] MinGW-w64 (mingw32-make) não encontrado no PATH!" -ForegroundColor Red
    Write-Host "    Você precisa instalar o MinGW-w64 (ex: via MSYS2) e adicioná-lo ao PATH do sistema." -ForegroundColor Yellow
    $missing = $true
} else {
    Write-Host "[OK] MinGW-w64 encontrado." -ForegroundColor Green
}

if ($missing) {
    Write-Host ""
    Write-Host "Por favor, instale as dependências faltantes, reinicie o terminal (para atualizar o PATH) e rode este script novamente." -ForegroundColor Red
    exit 1
}

# Obtém a localização do pacote GoCV armazenado em cache
Write-Host "`nBuscando diretório do módulo gocv.io/x/gocv..."
$goModOutput = go list -m -f '{{.Dir}}' gocv.io/x/gocv
if (-not $goModOutput) {
    Write-Host "Módulo GoCV não encontrado! Garantindo o download com 'go get -u gocv.io/x/gocv'..."
    go get -u gocv.io/x/gocv
    $goModOutput = go list -m -f '{{.Dir}}' gocv.io/x/gocv
}

if ($goModOutput -and (Test-Path $goModOutput)) {
    Write-Host "GoCV localizado em: $goModOutput" -ForegroundColor Green
    
    # Navega até a pasta do GoCV e executa o script de build oficial
    Set-Location $goModOutput
    
    if (Test-Path "win_build_opencv.cmd") {
        Write-Host "`nPreparando código-fonte do OpenCV..." -ForegroundColor Cyan
        
        $opencvDir = "C:\opencv"
        if (-not (Test-Path $opencvDir)) {
            New-Item -ItemType Directory -Path $opencvDir | Out-Null
        }

        $version = "4.13.0"
        
        if (-not (Test-Path "$opencvDir\opencv-$version")) {
            Write-Host "Baixando OpenCV $version..."
            Invoke-WebRequest -Uri "https://github.com/opencv/opencv/archive/refs/tags/$version.zip" -OutFile "$opencvDir\opencv.zip"
            Write-Host "Extraindo OpenCV..."
            Expand-Archive -Path "$opencvDir\opencv.zip" -DestinationPath $opencvDir -Force
            Remove-Item "$opencvDir\opencv.zip"
        }

        if (-not (Test-Path "$opencvDir\opencv_contrib-$version")) {
            Write-Host "Baixando OpenCV Contrib $version..."
            Invoke-WebRequest -Uri "https://github.com/opencv/opencv_contrib/archive/refs/tags/$version.zip" -OutFile "$opencvDir\opencv_contrib.zip"
            Write-Host "Extraindo OpenCV Contrib..."
            Expand-Archive -Path "$opencvDir\opencv_contrib.zip" -DestinationPath $opencvDir -Force
            Remove-Item "$opencvDir\opencv_contrib.zip"
        }

        Write-Host "`nIniciando compilação do OpenCV (win_build_opencv.cmd)..." -ForegroundColor Cyan
        Write-Host "Vá tomar um café, isso vai demorar (pode levar mais de 30 min)! ☕`n" -ForegroundColor Yellow
        
        # Executa o script nativo
        cmd.exe /c win_build_opencv.cmd
        
        Write-Host "`nProcesso finalizado!" -ForegroundColor Green
        Write-Host "Se o build foi um sucesso, você precisa adicionar C:\opencv\build\install\x64\mingw\bin ao seu PATH!" -ForegroundColor Yellow
        Write-Host "Agora você pode compilar o projeto executando 'go run .' na raiz do projeto (d:\dev\Webcam-go)." -ForegroundColor Cyan
    } else {
        Write-Host "Script 'win_build_opencv.cmd' não encontrado na pasta do GoCV." -ForegroundColor Red
    }
} else {
    Write-Host "Falha ao localizar o módulo GoCV no sistema." -ForegroundColor Red
}
