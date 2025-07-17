#!/bin/bash
set -e

TOOLS_DIR="tools"
MODELS_DIR="models"
YTDLP_BIN="$TOOLS_DIR/yt-dlp"
FFMPEG_BIN="$TOOLS_DIR/ffmpeg"
WHISPER_BIN="$TOOLS_DIR/whisper"

mkdir -p "$TOOLS_DIR"
mkdir -p "$MODELS_DIR"

# Detect OS and arch
OS=$(uname -s)
ARCH=$(uname -m)

# Function to check if ffmpeg is available
check_ffmpeg() {
    if command -v ffmpeg >/dev/null 2>&1; then
        echo "ffmpeg found in PATH: $(which ffmpeg)"
        return 0
    else
        return 1
    fi
}

# Function to install ffmpeg via package manager
install_ffmpeg_package() {
    echo "Attempting to install ffmpeg via package manager..."
    
    if [[ "$OS" == "Darwin" ]]; then
        if command -v brew >/dev/null 2>&1; then
            echo "Installing ffmpeg via Homebrew..."
            brew install ffmpeg
            return 0
        else
            echo "Homebrew not found. Please install Homebrew first: https://brew.sh/"
            return 1
        fi
    elif [[ "$OS" == "Linux" ]]; then
        if command -v apt-get >/dev/null 2>&1; then
            echo "Installing ffmpeg via apt-get..."
            sudo apt-get update && sudo apt-get install -y ffmpeg
            return 0
        elif command -v yum >/dev/null 2>&1; then
            echo "Installing ffmpeg via yum..."
            sudo yum install -y ffmpeg
            return 0
        elif command -v dnf >/dev/null 2>&1; then
            echo "Installing ffmpeg via dnf..."
            sudo dnf install -y ffmpeg
            return 0
        elif command -v apk >/dev/null 2>&1; then
            echo "Installing ffmpeg via apk..."
            sudo apk add ffmpeg
            return 0
        else
            echo "No supported package manager found (apt-get, yum, dnf, apk)"
            return 1
        fi
    else
        echo "Unsupported OS: $OS"
        return 1
    fi
}

# Download yt-dlp binary if not present
if [ ! -f "$YTDLP_BIN" ]; then
  echo "Downloading yt-dlp..."
  curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o "$YTDLP_BIN"
  chmod +x "$YTDLP_BIN"
else
  echo "yt-dlp already present."
fi

# Handle ffmpeg installation
if check_ffmpeg; then
    echo "ffmpeg already available in system PATH."
    # Create a symlink to the system ffmpeg for consistency
    ln -sf "$(which ffmpeg)" "$FFMPEG_BIN" 2>/dev/null || true
else
    echo "ffmpeg not found in PATH."
    
    # Try package manager installation
    if install_ffmpeg_package; then
        echo "ffmpeg installed successfully via package manager."
        # Create a symlink to the system ffmpeg
        ln -sf "$(which ffmpeg)" "$FFMPEG_BIN" 2>/dev/null || true
    else
        echo "Package manager installation failed. Attempting static download..."
        
        # Fall back to static download (existing logic)
        if [[ "$OS" == "Darwin" ]]; then
            # macOS: try multiple sources for ffmpeg
            if [[ "$ARCH" == "arm64" ]]; then
                # Try multiple sources for arm64
                FFMPEG_URLS=(
                    "https://evermeet.cx/ffmpeg/ffmpeg-arm64.zip"
                    "https://github.com/ffmpeg/ffmpeg/releases/download/n6.0/ffmpeg-6.0-macos64arm64-static.zip"
                )
            elif [[ "$ARCH" == "x86_64" ]]; then
                # Try multiple sources for x86_64
                FFMPEG_URLS=(
                    "https://evermeet.cx/ffmpeg/ffmpeg.zip"
                    "https://github.com/ffmpeg/ffmpeg/releases/download/n6.0/ffmpeg-6.0-macos64-static.zip"
                )
            else
                echo "Unsupported macOS architecture: $ARCH. Please install ffmpeg manually."
                exit 1
            fi
            
            # Try each URL until one works
            FFMPEG_DOWNLOADED=false
            for FFMPEG_URL in "${FFMPEG_URLS[@]}"; do
                echo "Trying ffmpeg URL: $FFMPEG_URL"
                if curl -L "$FFMPEG_URL" -o "$TOOLS_DIR/ffmpeg.zip" --fail --silent --show-error; then
                    if unzip -t "$TOOLS_DIR/ffmpeg.zip" > /dev/null 2>&1; then
                        unzip -o "$TOOLS_DIR/ffmpeg.zip" -d "$TOOLS_DIR/"
                        mv "$TOOLS_DIR/ffmpeg" "$FFMPEG_BIN" 2>/dev/null || true
                        rm "$TOOLS_DIR/ffmpeg.zip"
                        FFMPEG_DOWNLOADED=true
                        break
                    else
                        echo "Invalid zip file, trying next source..."
                        rm "$TOOLS_DIR/ffmpeg.zip"
                    fi
                else
                    echo "Download failed, trying next source..."
                fi
            done
            
            if [[ "$FFMPEG_DOWNLOADED" == "false" ]]; then
                echo "Failed to download ffmpeg from all sources."
                echo "Please install ffmpeg manually:"
                echo "  brew install ffmpeg"
                echo "  or download from https://ffmpeg.org/download.html"
                exit 1
            fi
        elif [[ "$OS" == "Linux" ]]; then
            # Linux: use BtbN/FFmpeg-Builds
            if [[ "$ARCH" == "x86_64" ]]; then
                # Get latest versioned tag from BtbN/FFmpeg-Builds (skip autobuilds, rc, latest)
                LATEST_TAG=$(curl -s https://api.github.com/repos/BtbN/FFmpeg-Builds/releases | grep '"tag_name":' | grep -o 'n[0-9.]*' | head -n 1)
                echo "Latest ffmpeg tag: $LATEST_TAG"
                FFMPEG_URL="https://github.com/BtbN/FFmpeg-Builds/releases/download/${LATEST_TAG}/ffmpeg-${LATEST_TAG}-latest-linux64-gpl-${LATEST_TAG}.tar.xz"
            else
                echo "Unsupported Linux architecture: $ARCH. Please install ffmpeg manually."
                exit 1
            fi
            curl -L "$FFMPEG_URL" -o "$TOOLS_DIR/ffmpeg.tar.xz"
            tar -xf "$TOOLS_DIR/ffmpeg.tar.xz" -C "$TOOLS_DIR/"
            FFMPEG_EXTRACTED=$(find "$TOOLS_DIR" -type f -name ffmpeg | head -n 1)
            mv "$FFMPEG_EXTRACTED" "$FFMPEG_BIN"
            rm "$TOOLS_DIR/ffmpeg.tar.xz"
        else
            echo "Unsupported OS: $OS. Please install ffmpeg manually and place it at $FFMPEG_BIN."
            exit 1
        fi
    fi
fi

# Compile whisper.cpp if not present
if [ ! -f "$WHISPER_BIN" ]; then
  echo "Compiling whisper.cpp..."
  if [ ! -d "whisper.cpp" ]; then
    git clone https://github.com/ggerganov/whisper.cpp.git
  fi
  cd whisper.cpp
  make
  cp build/bin/whisper-cli "../$WHISPER_BIN"
  cd ..
  chmod +x "$WHISPER_BIN"
  echo "whisper.cpp compiled successfully."
else
  echo "whisper.cpp already present."
fi

# Download default whisper model if not present
DEFAULT_MODEL="$MODELS_DIR/ggml-tiny.en.bin"
if [ ! -f "$DEFAULT_MODEL" ]; then
  echo "Downloading default whisper model (ggml-tiny.en.bin)..."
  curl -L https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.en.bin -o "$DEFAULT_MODEL"
  echo "Default model downloaded successfully."
else
  echo "Default whisper model already present."
fi

echo "Setup complete! All tools and models are ready."
echo "Available tools:"
echo "  - yt-dlp: $YTDLP_BIN"
echo "  - ffmpeg: $FFMPEG_BIN"
echo "  - whisper: $WHISPER_BIN"
echo "  - default model: $DEFAULT_MODEL" 