#!/bin/bash
# update-supabase-secret.sh - Script to update Supabase credentials in Kubernetes
# This script reads credentials from environment variables to avoid exposing them in command history

# Check if environment variables are set
if [ -z "$SUPABASE_URL" ] || [ -z "$SUPABASE_KEY" ]; then
    echo "Error: Required environment variables not set"
    echo "Please set the following environment variables:"
    echo "  export SUPABASE_URL=https://your-project.supabase.co"
    echo "  export SUPABASE_KEY=your-service-role-key"
    echo ""
    echo "For security, you can also add these to your ~/.bashrc or ~/.zshrc file"
    echo "or use a tool like direnv to load them automatically when in this directory."
    exit 1
fi

# Validate URL format
if [[ ! "$SUPABASE_URL" =~ ^https?://[a-zA-Z0-9.-]+\.supabase\.co/?$ ]]; then
    echo "Error: Supabase URL should be in format https://your-project.supabase.co"
    exit 1
fi

# Validate key format (basic check that it looks like a JWT)
if [[ ! "$SUPABASE_KEY" =~ ^ey ]]; then
    echo "Warning: Supabase key doesn't look like a valid service role key (should start with 'ey')"
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo "Updating Supabase credentials in Kubernetes..."

# Delete the existing secret if it exists
kubectl delete secret supabase --ignore-not-found

# Create a new secret with the provided values
kubectl create secret generic supabase \
  --from-literal=SUPABASE_URL="$SUPABASE_URL" \
  --from-literal=SUPABASE_KEY="$SUPABASE_KEY"

# Check if the secret was created successfully
if [ $? -eq 0 ]; then
    echo "Supabase secret updated successfully!"
    
    # Restart user-service to pick up the new credentials
    echo "Restarting user-service to apply new credentials..."
    kubectl rollout restart deployment user-service
    
    echo "Done! The user service should now be able to connect to Supabase."
    echo "You can check the status with: kubectl get pods"
else
    echo "Failed to update Supabase secret. Please check your kubectl configuration."
fi