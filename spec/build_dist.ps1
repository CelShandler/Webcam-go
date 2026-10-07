# build_dist.ps1
# Script para compilar o projeto e gerar o arquivo ZIP de distribuição.

Write-Host "=======================================================" -ForegroundColor Cyan
Write-Host "   Iniciando Build e Empacotamento do DVR Inteligente   " -ForegroundColor Cyan
Write-Host "=======================================================" -ForegroundColor Cyan
Write-Host ""

$exeName = "dvr_inteligente.exe"
$distDir = "dvr_inteligente_dist"
$zipName = "dvr_inteligente_dist.zip"
$openCvBinDir = "C:\opencv\build\install\x64\mingw\bin"

# 1. Compilando o executável Go
Write-Host "1. Compilando o executável Go..." -ForegroundColor Yellow
go build -ldflags="-H windowsgui" -o $exeName .
if ($LASTEXITCODE -ne 0) {
    Write-Host "[X] Erro na compilação do Go!" -ForegroundColor Red
    exit $LASTEXITCODE
}
Write-Host "[OK] Executável $exeName gerado com sucesso." -ForegroundColor Green

# 2. Criando o diretório de distribuição
Write-Host "`n2. Preparando pasta de distribuição ($distDir)..." -ForegroundColor Yellow
if (Test-Path $distDir) {
    Remove-Item -Recurse -Force $distDir
}
New-Item -ItemType Directory -Force -Path $distDir | Out-Null

# Move o executável
Move-Item -Path $exeName -Destination "$distDir\" -Force

# 3. Copiando pastas do frontend (static e templates)
Write-Host "`n3. Copiando arquivos estáticos e templates..." -ForegroundColor Yellow
if (Test-Path "static") {
    Copy-Item -Path "static" -Destination "$distDir\static" -Recurse -Force
}
if (Test-Path "templates") {
    Copy-Item -Path "templates" -Destination "$distDir\templates" -Recurse -Force
}
Write-Host "[OK] Pastas web copiadas." -ForegroundColor Green

# 4. Copiando as dependências do OpenCV
Write-Host "`n4. Copiando DLLs do OpenCV de $openCvBinDir..." -ForegroundColor Yellow
if (Test-Path $openCvBinDir) {
    # Copia as DLLs do OpenCV
    Copy-Item -Path "$openCvBinDir\libopencv_*.dll" -Destination "$distDir\" -Force -ErrorAction SilentlyContinue
    
    # Tenta copiar as DLLs do compilador MinGW que geralmente ficam juntas no build do OpenCV
    # ou em locais comuns do MinGW. Se não estiverem lá, o usuário pode precisar copiá-las manualmente da instalação do MSYS2/MinGW.
    Copy-Item -Path "$openCvBinDir\libstdc++-6.dll" -Destination "$distDir\" -Force -ErrorAction SilentlyContinue
    Copy-Item -Path "$openCvBinDir\libgcc_s_seh-1.dll" -Destination "$distDir\" -Force -ErrorAction SilentlyContinue
    Copy-Item -Path "$openCvBinDir\libwinpthread-1.dll" -Destination "$distDir\" -Force -ErrorAction SilentlyContinue
    Copy-Item -Path "C:\msys64\mingw64\bin\libstdc++-6.dll" -Destination "$distDir\" -Force -ErrorAction SilentlyContinue
    Copy-Item -Path "C:\msys64\mingw64\bin\libgcc_s_seh-1.dll" -Destination "$distDir\" -Force -ErrorAction SilentlyContinue
    Copy-Item -Path "C:\msys64\mingw64\bin\libwinpthread-1.dll" -Destination "$distDir\" -Force -ErrorAction SilentlyContinue

    Write-Host "[OK] DLLs do OpenCV copiadas." -ForegroundColor Green
} else {
    Write-Host "[!] AVISO: Diretório $openCvBinDir não encontrado. As DLLs do OpenCV não foram copiadas." -ForegroundColor Red
}

# 5. Compactando a pasta num arquivo ZIP
Write-Host "`n5. Gerando arquivo ZIP para distribuição ($zipName)..." -ForegroundColor Yellow
if (Test-Path $zipName) {
    Remove-Item -Force $zipName
}
Compress-Archive -Path "$distDir\*" -DestinationPath $zipName -Force

Write-Host "`n[OK] Arquivo ZIP gerado com sucesso: $zipName" -ForegroundColor Green
Write-Host "Você já pode enviar o arquivo '$zipName' para distribuição!" -ForegroundColor Cyan
