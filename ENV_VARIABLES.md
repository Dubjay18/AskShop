# Using Environment Variables with Kubernetes

## Why Use Environment Variables?

Using environment variables for sensitive information like API keys and secrets provides several advantages:

1. **Security**: Your credentials aren't committed to your repository
2. **Flexibility**: Different developers can use different credentials
3. **DevOps Best Practice**: Follows the principle of separating code from configuration

## Setup Instructions

### 1. Create Your .env File

```bash
# Copy the example file
cp .env.example .env

# Edit with your actual credentials
nano .env
```

Your `.env` file should contain:

```
SUPABASE_URL=https://your-actual-project.supabase.co
SUPABASE_KEY=your-actual-service-role-key
```

### 2. Load Credentials into Kubernetes

```bash
# Source your environment variables
source .env

# Run the script to update Kubernetes secrets
./scripts/load-env-to-k8s.sh
```

This script:
- Creates/updates the `supabase` Kubernetes secret with your environment variables
- Restarts the user service to apply the changes

### 3. Verify the Secret was Created

```bash
kubectl get secret supabase -o yaml
```

The output will show your credentials are securely stored in Kubernetes.

## Automating with direnv (Optional)

For convenience, you can use [direnv](https://direnv.net/) to automatically load environment variables when you enter your project directory:

1. Install direnv:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install direnv
   
   # macOS
   brew install direnv
   ```

2. Create a `.envrc` file:
   ```bash
   echo 'source_env .env' > .envrc
   direnv allow
   ```

3. Now environment variables will load automatically when you enter the directory.

## Important Notes

- **Never commit** your `.env` file to git
- Make sure `.env` is in your `.gitignore`
- Use different environment variables for development, staging, and production

## Troubleshooting

If you encounter issues:

1. Verify your environment variables are set:
   ```bash
   echo $SUPABASE_URL
   ```

2. Check the Kubernetes secret:
   ```bash
   kubectl describe secret supabase
   ```

3. Check the user service logs:
   ```bash
   kubectl logs -f deployment/user-service
   ```