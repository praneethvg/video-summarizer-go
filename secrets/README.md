# Secrets Directory

Place your secret files here:

- `gdrive_credentials.json` - Google Drive OAuth credentials
- `gdrive_token.json` - Google Drive OAuth token
- `oauth_client_secret.json` - OAuth client secret (if using)

## File Permissions

Ensure these files have restricted permissions:
```bash
chmod 600 secrets/*
```

## Security Notes

- Never commit these files to version control
- Use appropriate secret management in production
- Rotate credentials regularly
