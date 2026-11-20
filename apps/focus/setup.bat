@echo off

REM Setup script for AI Focus application on Windows

echo 🔧 Setting up AI Focus Assistant...

REM Check if Go is installed
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Go is not installed. Please install Go 1.21+ from https://golang.org/dl/
    exit /b 1
)

echo ✅ Go is installed

REM Check if Ollama is installed
where ollama >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Ollama is not installed. Please install Ollama from https://ollama.ai/
    exit /b 1
)

echo ✅ Ollama is installed

REM Pull the local AI model (optional; the suite works offline without it)
set "pull_model="
set /p pull_model=📥 Pull the llama3 model with Ollama now? (y/N): 
if /i "%pull_model%"=="y" (
    echo 📥 Pulling llama3 model...
    ollama pull llama3
) else (
    echo ⏭️  Skipping the model pull. AI features will use whatever models Ollama already has.
)

REM Install Go dependencies
echo 📦 Installing Go dependencies...
go mod tidy

echo ✅ Setup complete!
echo.
echo To build the application, run:
echo   go build -o focus.exe cmd/focus/main.go
echo.
echo To run directly, use:
echo   go run cmd/focus/main.go