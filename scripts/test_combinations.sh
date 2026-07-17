#!/bin/bash
# test_combinations.sh
# Tests generating and compiling projects for all combinations of ginboot-cli flags

set -e

# Default to the sibling ginboot framework directory
CLI_DIR=$(pwd)
FRAMEWORK_DIR=${1:-"../ginboot"}

echo "Building ginboot-cli..."
go build -o bin/ginboot-cli main.go
CLI_BIN=$(pwd)/bin/ginboot-cli

if [ ! -d "$FRAMEWORK_DIR" ]; then
    echo "Warning: Local ginboot framework not found at $FRAMEWORK_DIR. Tests will pull from remote."
    USE_LOCAL=false
else
    # Resolve absolute path
    FRAMEWORK_DIR=$(cd "$FRAMEWORK_DIR" && pwd)
    USE_LOCAL=true
    echo "Using local framework at: $FRAMEWORK_DIR"
fi

DBS=("none" "mongodb" "postgres" "mysql" "dynamodb")
DEPLOYS=("http" "lambda")
STORAGES=("none" "s3")

TEST_DIR="/tmp/ginboot_cli_tests_$$"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"

success_count=0
fail_count=0
failed_combos=()

for db in "${DBS[@]}"; do
  for deploy in "${DEPLOYS[@]}"; do
    for storage in "${STORAGES[@]}"; do
      project_name="test${db}${deploy}${storage}"
      echo "------------------------------------------------------"
      echo "Testing combination: DB=$db, Deploy=$deploy, Storage=$storage"
      
      cd "$TEST_DIR"
      
      # Generate project
      "$CLI_BIN" new "$project_name" --module "github.com/test/$project_name" --db "$db" --deploy "$deploy" --storage "$storage" > /dev/null
      
      cd "$project_name"
      
      if [ "$USE_LOCAL" = true ]; then
          go work init
          go work use .
          go work use "$FRAMEWORK_DIR"
          
          # Add database modules
          if [ -d "$FRAMEWORK_DIR/db/inmemory" ]; then go work use "$FRAMEWORK_DIR/db/inmemory"; fi
          if [ -d "$FRAMEWORK_DIR/db/mongo" ]; then go work use "$FRAMEWORK_DIR/db/mongo"; fi
          if [ -d "$FRAMEWORK_DIR/db/sql" ]; then go work use "$FRAMEWORK_DIR/db/sql"; fi
          if [ -d "$FRAMEWORK_DIR/db/dynamodb" ]; then go work use "$FRAMEWORK_DIR/db/dynamodb"; fi
          
          # Add storage modules
          if [ -d "$FRAMEWORK_DIR/storage/s3" ]; then go work use "$FRAMEWORK_DIR/storage/s3"; fi
          
          # Add runtime modules
          if [ -d "$FRAMEWORK_DIR/runtime/lambda" ]; then go work use "$FRAMEWORK_DIR/runtime/lambda"; fi
      fi
      
      go mod tidy > /dev/null 2>&1
      
      if go build -o /dev/null; then
        echo "✅ SUCCESS"
        success_count=$((success_count + 1))
      else
        echo "❌ FAILED"
        fail_count=$((fail_count + 1))
        failed_combos+=("$project_name")
      fi
      
    done
  done
done

echo "======================================================"
echo "Test Summary: $success_count succeeded, $fail_count failed."
if [ $fail_count -gt 0 ]; then
  echo "Failed combinations:"
  for combo in "${failed_combos[@]}"; do
    echo "  - $combo"
  done
  exit 1
fi
