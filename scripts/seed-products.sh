#!/bin/bash

# Build and run the product seeder
echo "Building product seeder..."
cd /home/jen/Documents/projects/AskShop

# Build the seeder
go build -o build/product-seeder ./services/product-service/cmd/seed/

# Run the seeder
echo "Running product seeder..."
./build/product-seeder "$@"
