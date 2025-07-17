# Google Drive Access Setup

This guide explains how to set up Google Drive access for the video summarizer using either OAuth client authentication (recommended for user accounts) or a Google service account (for server-to-server automation).

---

## 1. OAuth Client Authentication (User Account)

**Recommended for most users. Allows uploading to your own Google Drive.**

### Steps:

1. **Go to the [Google Cloud Console](https://console.cloud.google.com/apis/credentials)**
2. **Create a new OAuth 2.0 Client ID**
   - Application type: Desktop app
   - Download the `client_secret.json` (rename to `oauth_client_secret.json` in your project root)
3. **Enable the Google Drive API** for your project
4. **Run the provided CLI tool to generate a refresh token:**
   ```sh
   go run cmd/gdrive-auth/main.go --client-secret oauth_client_secret.json --output gdrive_token.json
   ```
   - Follow the instructions to authenticate in your browser and paste the code.
   - This will create `gdrive_token.json`.
5. **Configure your `config.yaml` (engine config file):**
   ```yaml
   output_provider: gdrive
   gdrive_auth_method: oauth
   gdrive_credentials_file: oauth_client_secret.json
   gdrive_token_file: gdrive_token.json
   gdrive_folder_id: <your-folder-id>  # Optional: upload destination
   ```

---

## 2. Service Account Authentication (Server-to-Server)

**Use for automation or uploading to a shared/team drive.**

### Steps:

1. **Go to the [Google Cloud Console](https://console.cloud.google.com/apis/credentials)**
2. **Create a new Service Account**
   - Download the JSON key file (rename to `service_account.json` in your project root)
3. **Enable the Google Drive API** for your project
4. **Share the target Google Drive folder with the service account email**
   - Find the service account email in the JSON file
   - Share the folder as you would with any user
5. **Configure your `config.yaml` (engine config file):**
   ```yaml
   output_provider: gdrive
   gdrive_auth_method: service_account
   gdrive_credentials_file: service_account.json
   gdrive_folder_id: <your-folder-id>
   ```

---

## Notes
- Never commit credential or token files to git (they are gitignored by default).
- For more details, see the [Google Drive API docs](https://developers.google.com/drive/api/v3/quickstart/go).
- If you need to reset authentication, delete the token file and re-run the auth tool. 