#!/bin/bash

# Set colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}===== Supabase Connection Diagnostic Tool =====${NC}"
echo

# Check if SUPABASE_URL is set
if [ -z "$SUPABASE_URL" ]; then
    # Try to load from .env file
    if [ -f .env ]; then
        echo -e "${YELLOW}Loading environment variables from .env file...${NC}"
        export $(grep -v '^#' .env | xargs)
    else
        echo -e "${RED}Error: SUPABASE_URL environment variable is not set and .env file not found${NC}"
        exit 1
    fi
fi

# Extract domain from SUPABASE_URL
if [[ "$SUPABASE_URL" =~ ^https?://([^/]+) ]]; then
    DOMAIN="${BASH_REMATCH[1]}"
    echo -e "${GREEN}Found Supabase domain: $DOMAIN${NC}"
else
    echo -e "${RED}Error: Could not extract domain from SUPABASE_URL: $SUPABASE_URL${NC}"
    exit 1
fi

echo -e "\n${YELLOW}===== Testing DNS Resolution =====${NC}"

# Try to resolve the domain using host command
echo -e "Testing DNS resolution with 'host' command..."
HOST_RESULT=$(host "$DOMAIN" 2>&1)
HOST_STATUS=$?

if [ $HOST_STATUS -eq 0 ]; then
    echo -e "${GREEN}✓ DNS resolution successful${NC}"
    
    # Extract IP address
    if [[ "$HOST_RESULT" =~ has\ address\ ([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+) ]]; then
        IP="${BASH_REMATCH[1]}"
        echo -e "${GREEN}Found IP address: $IP${NC}"
        
        # Ask if user wants to update the .env file
        read -p "Do you want to add this IP to your .env file as SUPABASE_IP? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            # Check if .env exists
            if [ -f .env ]; then
                # Check if SUPABASE_IP already exists
                if grep -q "SUPABASE_IP=" .env; then
                    # Update existing entry
                    sed -i "s/SUPABASE_IP=.*/SUPABASE_IP=$IP/" .env
                else
                    # Add new entry
                    echo "SUPABASE_IP=$IP" >> .env
                fi
                echo -e "${GREEN}Successfully updated .env file with SUPABASE_IP=$IP${NC}"
            else
                echo "SUPABASE_IP=$IP" > .env
                echo -e "${GREEN}Created .env file with SUPABASE_IP=$IP${NC}"
            fi
            
            # Ask if user wants to update Kubernetes secrets
            read -p "Do you want to update Kubernetes secrets with this IP? (y/n) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                if [ -f ./scripts/update-k8s-secrets.sh ]; then
                    echo -e "${YELLOW}Running update-k8s-secrets.sh...${NC}"
                    ./scripts/update-k8s-secrets.sh
                else
                    echo -e "${RED}Error: update-k8s-secrets.sh script not found${NC}"
                fi
            fi
        fi
    else
        echo -e "${YELLOW}Warning: Could not extract IP address from host command output${NC}"
        echo "$HOST_RESULT"
    fi
else
    echo -e "${RED}✗ DNS resolution failed${NC}"
    echo "$HOST_RESULT"
    
    # Try using dig command as a fallback
    echo -e "\n${YELLOW}Trying alternative DNS resolution with 'dig' command...${NC}"
    if command -v dig &> /dev/null; then
        DIG_RESULT=$(dig "$DOMAIN" +short)
        if [ -n "$DIG_RESULT" ]; then
            echo -e "${GREEN}Found IP address(es) using dig:${NC}"
            echo "$DIG_RESULT"
            
            # Use the first IP address
            IP=$(echo "$DIG_RESULT" | head -1)
            
            # Ask if user wants to update the .env file
            read -p "Do you want to add the first IP ($IP) to your .env file as SUPABASE_IP? (y/n) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                if [ -f .env ]; then
                    if grep -q "SUPABASE_IP=" .env; then
                        sed -i "s/SUPABASE_IP=.*/SUPABASE_IP=$IP/" .env
                    else
                        echo "SUPABASE_IP=$IP" >> .env
                    fi
                    echo -e "${GREEN}Successfully updated .env file with SUPABASE_IP=$IP${NC}"
                else
                    echo "SUPABASE_IP=$IP" > .env
                    echo -e "${GREEN}Created .env file with SUPABASE_IP=$IP${NC}"
                fi
                
                # Ask if user wants to update Kubernetes secrets
                read -p "Do you want to update Kubernetes secrets with this IP? (y/n) " -n 1 -r
                echo
                if [[ $REPLY =~ ^[Yy]$ ]]; then
                    if [ -f ./scripts/update-k8s-secrets.sh ]; then
                        echo -e "${YELLOW}Running update-k8s-secrets.sh...${NC}"
                        ./scripts/update-k8s-secrets.sh
                    else
                        echo -e "${RED}Error: update-k8s-secrets.sh script not found${NC}"
                    fi
                fi
            fi
        else
            echo -e "${RED}✗ DNS resolution with dig also failed${NC}"
        fi
    else
        echo -e "${YELLOW}dig command not available. Consider installing it for better DNS diagnostics${NC}"
    fi
fi

echo -e "\n${YELLOW}===== Testing HTTP Connection =====${NC}"

# Try to connect to the SUPABASE_URL
echo -e "Testing HTTP connection to $SUPABASE_URL..."
CURL_RESULT=$(curl -s -o /dev/null -w "%{http_code}" -m 10 "$SUPABASE_URL")
CURL_STATUS=$?

if [ $CURL_STATUS -eq 0 ]; then
    echo -e "${GREEN}✓ HTTP connection successful (HTTP status: $CURL_RESULT)${NC}"
else
    echo -e "${RED}✗ HTTP connection failed${NC}"
    
    # If we have an IP, try connecting directly to it
    if [ -n "$IP" ]; then
        # Construct URL with IP instead of domain
        if [[ "$SUPABASE_URL" =~ ^(https?)://[^/]+(/.*)$ ]]; then
            PROTO="${BASH_REMATCH[1]}"
            PATH_PART="${BASH_REMATCH[2]}"
            IP_URL="$PROTO://$IP$PATH_PART"
            
            echo -e "\n${YELLOW}Trying direct IP connection: $IP_URL${NC}"
            CURL_IP_RESULT=$(curl -s -o /dev/null -w "%{http_code}" -m 10 -H "Host: $DOMAIN" "$IP_URL")
            CURL_IP_STATUS=$?
            
            if [ $CURL_IP_STATUS -eq 0 ]; then
                echo -e "${GREEN}✓ Direct IP connection successful (HTTP status: $CURL_IP_RESULT)${NC}"
                echo -e "${GREEN}This confirms the IP-based connection approach should work!${NC}"
            else
                echo -e "${RED}✗ Direct IP connection also failed${NC}"
                echo -e "${YELLOW}This might indicate network connectivity issues beyond DNS resolution.${NC}"
            fi
        else
            echo -e "${YELLOW}Could not parse URL for direct IP test${NC}"
        fi
    fi
fi

echo -e "\n${YELLOW}===== Kubernetes DNS Check =====${NC}"
echo -e "To check DNS resolution from inside the Kubernetes cluster, run:"
echo -e "${GREEN}kubectl run -i --tty --rm debug --image=tutum/dnsutils --restart=Never -- bash${NC}"
echo -e "Then inside the pod, run:"
echo -e "${GREEN}nslookup $DOMAIN${NC}"

echo -e "\n${YELLOW}===== Next Steps =====${NC}"
echo -e "1. If you added SUPABASE_IP to your .env file, restart the user service:"
echo -e "   ${GREEN}kubectl rollout restart deployment user-service${NC}"
echo -e "2. Check the logs to see if the connection is successful:"
echo -e "   ${GREEN}kubectl logs -f deployment/user-service${NC}"
echo -e "3. For more information, refer to SUPABASE_CONNECTIVITY.md"

echo -e "\n${GREEN}Diagnostic complete!${NC}"