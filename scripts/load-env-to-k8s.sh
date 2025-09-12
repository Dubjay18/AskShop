#!/bin/bash
# load-env-to-k8s.sh - A secure way to load environment variables into Kubernetes secrets
# Usage: source .env && ./scripts/load-env-to-k8s.sh

# Add debugging info to see what's happening
echo "Checking environment variables..."
echo "SUPABASE_URL is ${SUPABASE_URL:-(not set)}"
echo "SUPABASE_KEY is ${SUPABASE_KEY:+set but value not shown for security}"
[ -z "$SUPABASE_KEY" ] && echo "SUPABASE_KEY is not set"

# Check if SUPABASE_URL and SUPABASE_KEY are set
if [ -z "$SUPABASE_URL" ] || [ -z "$SUPABASE_KEY" ]; then
    echo "Error: Required environment variables not set"
    echo "Please run: source .env"
    echo "Make sure your .env file contains SUPABASE_URL and SUPABASE_KEY"
    exit 1
fi

# Update the secret in Kubernetes
echo "Updating Supabase secret with values from environment variables..."
kubectl delete secret supabase --ignore-not-found
kubectl create secret generic supabase \
  --from-literal=SUPABASE_URL="$SUPABASE_URL" \
  --from-literal=SUPABASE_KEY="$SUPABASE_KEY"

# Check if the secret was created successfully
if [ $? -eq 0 ]; then
    echo "✅ Supabase secret updated successfully!"
    
    # Restart user-service to pick up the new credentials
    echo "Restarting user-service to apply new credentials..."
    kubectl rollout restart deployment user-service
    
    echo "Done! The user service should now be able to connect to Supabase."
    echo "You can check the status with: kubectl get pods"
else
    echo "❌ Failed to update Supabase secret. Please check your kubectl configuration."
fi