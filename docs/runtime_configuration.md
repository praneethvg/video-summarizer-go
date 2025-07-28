# Runtime Configuration Guide

This document explains how to configure the Video Summarizer application at runtime for different deployment environments.

## Configuration Hierarchy

The application uses a three-tier configuration hierarchy:

1. **Environment Variables** (highest priority)
2. **YAML Configuration Files** (medium priority)
3. **Default Values** (lowest priority)

## Environment Variable Naming Convention

All environment variables use the `VS_` prefix followed by the configuration path:

- `VS_<SECTION>_<KEY>` for simple values
- `VS_<SECTION>_<SUBSECTION>_<KEY>` for nested values

## Core Configuration Variables

### OpenAI Settings
```bash
VS_OPENAI_API_KEY=sk-your-openai-key-here
VS_OPENAI_MODEL=gpt-4o
VS_OPENAI_MAX_TOKENS=10000
```

### Google Drive Settings
```bash
VS_GDRIVE_AUTH_METHOD=oauth
VS_GDRIVE_CREDENTIALS_FILE=/app/secrets/gdrive_credentials.json
VS_GDRIVE_TOKEN_FILE=/app/secrets/gdrive_token.json
VS_GDRIVE_FOLDER_ID=your-folder-id
VS_UPLOAD_SUMMARY=true
VS_UPLOAD_TRANSCRIPT=true
```

### Server Settings
```bash
VS_SERVER_PORT=8080
VS_SERVER_HOST=0.0.0.0
```

### Paths and Directories
```bash
VS_ENGINE_CONFIG_PATH=/app/config/config.yaml
VS_PROMPTS_DIR=/app/prompts
VS_TMP_DIR=/tmp
VS_YT_DLP_PATH=/app/tools/yt-dlp
VS_WHISPER_PATH=/app/tools/whisper
VS_WHISPER_MODEL_PATH=/app/models/ggml-tiny.en.bin
```

### Concurrency Settings
```bash
VS_CONCURRENCY_TRANSCRIPTION=2
VS_CONCURRENCY_SUMMARIZATION=3
VS_CONCURRENCY_VIDEO_INFO=1
VS_CONCURRENCY_OUTPUT=1
VS_CONCURRENCY_CLEANUP=1
VS_CONCURRENCY_AUDIO_DOWNLOAD=1
```

## Background Sources Configuration

Background sources are configured via the separate `sources.yaml` file. For runtime configuration:

### Docker: Mount Different Sources Files
```bash
# Development sources
docker run -d \
  --name video-summarizer \
  -p 8080:8080 \
  -v /path/to/sources-dev.yaml:/app/sources.yaml:ro \
  -v /path/to/secrets:/app/secrets:ro \
  video-summarizer:latest

# Production sources
docker run -d \
  --name video-summarizer \
  -p 8080:8080 \
  -v /path/to/sources-prod.yaml:/app/sources.yaml:ro \
  -v /path/to/secrets:/app/secrets:ro \
  video-summarizer:latest

# Disable all sources
docker run -d \
  --name video-summarizer \
  -p 8080:8080 \
  -v /path/to/sources-empty.yaml:/app/sources.yaml:ro \
  -v /path/to/secrets:/app/secrets:ro \
  video-summarizer:latest
```

### Kubernetes: Use ConfigMaps
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: video-summarizer-sources-config
data:
  sources.yaml: |
    sources:
      - name: "tech_tutorials"
        type: "youtube_search"
        enabled: true
        interval: "1h"
        prompt_id: "educational"
        category: "education"
        config:
          queries:
            - "machine learning tutorials"
          max_videos_per_run: 5
```

### Environment Variable Override
You can also override the sources config path:
```bash
VS_SOURCES_CONFIG_PATH=/app/custom-sources.yaml
```

## Docker Deployment

### 1. Environment Variables Approach

```bash
docker run -d \
  --name video-summarizer \
  -p 8080:8080 \
  -e VS_OPENAI_API_KEY=sk-your-key \
  -e VS_GDRIVE_FOLDER_ID=your-folder-id \
  -e VS_SERVER_PORT=8080 \
  -v /path/to/secrets:/app/secrets:ro \
  video-summarizer:latest
```

### 2. Configuration File Mounting

```bash
docker run -d \
  --name video-summarizer \
  -p 8080:8080 \
  -v /path/to/config.yaml:/app/config/config.yaml:ro \
  -v /path/to/service.yaml:/app/service.yaml:ro \
  -v /path/to/secrets:/app/secrets:ro \
  video-summarizer:latest
```

### 3. Docker Compose Example

```yaml
version: '3.8'
services:
  video-summarizer:
    image: video-summarizer:latest
    ports:
      - "8080:8080"
    environment:
      - VS_OPENAI_API_KEY=${OPENAI_API_KEY}
      - VS_GDRIVE_FOLDER_ID=${GDRIVE_FOLDER_ID}
      - VS_SERVER_PORT=8080
    volumes:
      - ./secrets:/app/secrets:ro
      - ./config:/app/config:ro
      - ./logs:/app/logs
    restart: unless-stopped
```

## Kubernetes Deployment

### 1. ConfigMap for Non-Sensitive Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: video-summarizer-config
data:
  config.yaml: |
    summarizer_provider: openai
    openai_model: gpt-4o
    openai_max_tokens: 10000
    output_provider: gdrive
    gdrive_auth_method: oauth
    upload_summary: true
    upload_transcript: true
    concurrency:
      transcription: 2
      summarization: 3
      video_info: 1
      output: 1
      cleanup: 1
      audio_download: 1

  service.yaml: |
    server:
      port: 8080
      host: 0.0.0.0
    engine_config_path: /app/config/config.yaml
    prompts_dir: /app/prompts
    background_sources:
      sources:
        - name: tech_tutorials
          type: youtube_search
          enabled: true
          interval: 30m
          prompt_id: educational
          category: education
          config:
            queries:
              - machine learning tutorials
              - Go programming tips
            max_videos_per_run: 5
            channel_videos_lookback: 50
```

### 2. Secret for Sensitive Data

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: video-summarizer-secrets
type: Opaque
data:
  openai-api-key: <base64-encoded-openai-key>
  gdrive-credentials.json: <base64-encoded-credentials>
  gdrive-token.json: <base64-encoded-token>
```

### 3. Deployment with Volume Mounts

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: video-summarizer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: video-summarizer
  template:
    metadata:
      labels:
        app: video-summarizer
    spec:
      containers:
      - name: video-summarizer
        image: video-summarizer:latest
        ports:
        - containerPort: 8080
        env:
        - name: VS_OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: video-summarizer-secrets
              key: openai-api-key
        - name: VS_GDRIVE_CREDENTIALS_FILE
          value: /app/secrets/gdrive_credentials.json
        - name: VS_GDRIVE_TOKEN_FILE
          value: /app/secrets/gdrive_token.json
        volumeMounts:
        - name: config-volume
          mountPath: /app/config
        - name: secrets-volume
          mountPath: /app/secrets
          readOnly: true
        - name: logs-volume
          mountPath: /app/logs
      volumes:
      - name: config-volume
        configMap:
          name: video-summarizer-config
      - name: secrets-volume
        secret:
          secretName: video-summarizer-secrets
      - name: logs-volume
        persistentVolumeClaim:
          claimName: video-summarizer-logs-pvc
```

### 4. Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: video-summarizer-service
spec:
  selector:
    app: video-summarizer
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

## Secret Management Best Practices

### 1. Never Bake Secrets into Images
- ✅ Use environment variables or mounted files
- ❌ Don't include secrets in Docker images

### 2. Use Appropriate Secret Management
- **Docker**: Environment variables or mounted files
- **Kubernetes**: Kubernetes Secrets
- **Cloud**: Cloud Secret Managers (AWS Secrets Manager, GCP Secret Manager, Azure Key Vault)

### 3. File Permissions
Ensure secret files have appropriate permissions:
```bash
chmod 600 /path/to/secrets/*
```

### 4. Rotation Strategy
- Implement secret rotation procedures
- Use short-lived tokens where possible
- Monitor for expired credentials

## Configuration Validation

The application validates configuration at startup:

1. **Required Fields**: Checks for essential configuration
2. **File Existence**: Validates that referenced files exist
3. **Permission Checks**: Ensures proper file permissions
4. **Connection Tests**: Tests external service connectivity

## Troubleshooting

### Common Issues

1. **Missing Environment Variables**
   - Check that all required environment variables are set
   - Verify naming convention (`VS_` prefix)

2. **File Permission Errors**
   - Ensure secret files have correct permissions
   - Check volume mount paths

3. **Configuration Conflicts**
   - Environment variables override YAML config
   - Check for conflicting values

### Debug Mode

Enable debug logging to troubleshoot configuration issues:
```bash
VS_LOG_LEVEL=debug
```

## Migration from File-Based to Runtime Configuration

1. **Identify Current Configuration**: List all hardcoded values
2. **Create Environment Variables**: Convert to environment variable format
3. **Update Deployment Scripts**: Modify Docker/Kubernetes manifests
4. **Test Configuration**: Verify all settings work correctly
5. **Document Changes**: Update team documentation 