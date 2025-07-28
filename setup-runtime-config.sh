#!/bin/bash

# Setup script for runtime configuration
# This script creates the necessary directories and files for runtime configuration

set -e

echo "Setting up runtime configuration for Video Summarizer..."

# Create directories
echo "Creating directories..."
mkdir -p secrets config logs

# Set proper permissions
echo "Setting permissions..."
chmod 700 secrets
chmod 755 config logs

# Create .env template if it doesn't exist
if [ ! -f .env ]; then
    echo "Creating .env template..."
    cat > .env << EOF
# Video Summarizer Runtime Configuration
# Copy this file to .env and fill in your actual values

# OpenAI Configuration
OPENAI_API_KEY=sk-your-openai-api-key-here

# Google Drive Configuration
GDRIVE_FOLDER_ID=your-google-drive-folder-id

# Optional: Override other settings
# VS_OPENAI_MODEL=gpt-4o
# VS_SERVER_PORT=8080
# VS_CONCURRENCY_TRANSCRIPTION=2
EOF
    echo "Created .env template. Please edit it with your actual values."
else
    echo ".env file already exists."
fi

# Create secrets directory structure
echo "Setting up secrets directory..."
cat > secrets/README.md << EOF
# Secrets Directory

Place your secret files here:

- \`gdrive_credentials.json\` - Google Drive OAuth credentials
- \`gdrive_token.json\` - Google Drive OAuth token
- \`oauth_client_secret.json\` - OAuth client secret (if using)

## File Permissions

Ensure these files have restricted permissions:
\`\`\`bash
chmod 600 secrets/*
\`\`\`

## Security Notes

- Never commit these files to version control
- Use appropriate secret management in production
- Rotate credentials regularly
EOF

# Create config directory structure
echo "Setting up config directory..."
[ ! -f config/config.yaml ] && cp config.yaml.template config/config.yaml
[ ! -f config/service.yaml ] && cp service.yaml.template config/service.yaml
[ ! -f config/sources.yaml ] && cp sources.yaml.template config/sources.yaml

echo "Configuration setup complete!"
echo ""
echo "Next steps:"
echo "1. Edit .env file with your actual API keys and settings"
echo "2. Place your secret files in the secrets/ directory"
echo "3. Customize config/config.yaml and config/service.yaml if needed"
echo "4. Build the application: ./build_all.sh"
echo "5. Run: docker-compose -f docker-compose.example.yml up"
echo ""
echo "Available scripts:"
echo "- ./build_all.sh - Build all binaries"
echo "- ./setup_tools.sh - Install external tools"
echo "- ./setup-runtime-config.sh - This script"
echo ""
echo "For Kubernetes deployment, see docs/runtime_configuration.md" 