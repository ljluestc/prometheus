#!/bin/bash
# Script to generate the promql parser

# Ensure goyacc is installed
command -v goyacc >/dev/null 2>&1 || {
  echo "Installing goyacc..."
  go install golang.org/x/tools/cmd/goyacc@latest
}

# Generate the parser
echo "Generating promql parser..."
goyacc -o promql/parser/generated_parser.y.go promql/parser/generated_parser.y

# Make the script executable
chmod +x gen_parser.sh

echo "Parser generation complete."
