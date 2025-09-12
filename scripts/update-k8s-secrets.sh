#!/bin/bash
# update-k8s-secrets.sh - Combined script to load .env and update Kubernetes secrets

# Define colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting Kubernetes secret update process...${NC}"

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${RED}Error: .env file not found!${NC}"
    echo "Please create a .env file with your Supabase credentials:"
    echo "SUPABASE_URL=https://your-project.supabase.co"
    echo "SUPABASE_KEY=your-service-role-key"
    exit 1
fi

# Export variables from .env file (making them available to this script)
echo "Loading environment variables from .env file..."
export $(grep -v '^#' .env | xargs)

# Check if variables are now set
echo "Checking if variables were loaded correctly..."
if [ -z "$SUPABASE_URL" ]; then
    echo -e "${RED}Error: SUPABASE_URL not found in .env file or is empty${NC}"
    exit 1
fi

if [ -z "$SUPABASE_KEY" ]; then
    echo -e "${RED}Error: SUPABASE_KEY not found in .env file or is empty${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Environment variables loaded successfully${NC}"
echo "SUPABASE_URL is set to: ${SUPABASE_URL}"
echo "SUPABASE_KEY is: [hidden for security]"

# Update the Kubernetes secret
echo -e "\n${YELLOW}Updating Kubernetes secret...${NC}"
kubectl delete secret supabase --ignore-not-found
kubectl create secret generic supabase \
  --from-literal=SUPABASE_URL="$SUPABASE_URL" \
  --from-literal=SUPABASE_KEY="$SUPABASE_KEY"

# Check if the secret was created successfully
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Supabase secret updated successfully!${NC}"
    
    # Restart user-service to pick up the new credentials
    echo -e "\n${YELLOW}Restarting user-service to apply new credentials...${NC}"
    kubectl rollout restart deployment user-service
    
    echo -e "${GREEN}Done! The user service should now be able to connect to Supabase.${NC}"
    echo "You can check the status with: kubectl get pods"
else
    echo -e "${RED}❌ Failed to update Supabase secret. Please check your kubectl configuration.${NC}"
    exit 1
fi