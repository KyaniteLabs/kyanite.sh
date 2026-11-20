#!/bin/bash

# Setup script for AI Focus application

echo "🔧 Setting up AI Focus Assistant..."

# Check if Go is installed
if ! command -v go &> /dev/null
then
    echo "❌ Go is not installed. Please install Go 1.21+ from https://golang.org/dl/"
    exit 1
fi

echo "✅ Go is installed"

# Check if Ollama is installed
if ! command -v ollama &> /dev/null
then
    echo "❌ Ollama is not installed. Please install Ollama from https://ollama.ai/"
    exit 1
fi

echo "✅ Ollama is installed"

# Pull the local AI model (optional; the suite works offline without it)
read -p "📥 Pull the llama3 model with Ollama now? (y/N): " pull_model
pull_model="${pull_model,,}"
if [ "$pull_model" = "y" ] || [ "$pull_model" = "yes" ]; then
    echo "📥 Pulling llama3 model..."
    ollama pull llama3
else
    echo "⏭️  Skipping the model pull. AI features will use whatever models Ollama already has."
fi

# Install Go dependencies
echo "📦 Installing Go dependencies..."
go mod tidy

echo "✅ Setup complete!"
echo ""
echo "To build the application, run:"
echo "  go build -o focus cmd/focus/main.go"
echo ""
echo "To run directly, use:"
echo "  go run cmd/focus/main.go"