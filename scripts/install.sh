#!/bin/bash

set -e

LOG_START='\n\e[1;36m'  # new line + bold + color
LOG_END='\n\e[0m'       # new line + reset color
DONE_START='\n\e[1;32m' # new line + bold + green
DONE_END='\n\n\e[0m'    # new line + reset

WORKDIR=$PWD

# Organize directories.

printf "${LOG_START}Organizing directories...${LOG_END}"

if [ -d "$WORKDIR/pkg/chain/gen" ]; then rm -rf "$WORKDIR/pkg/chain/gen"; fi
if [ -d "$WORKDIR/temporary" ]; then rm -rf "$WORKDIR/temporary"; fi

mkdir -p "$WORKDIR/pkg/chain/gen/core/abi"
mkdir -p "$WORKDIR/pkg/chain/gen/ecdsa/abi"
mkdir -p "$WORKDIR/pkg/chain/gen/erc20/abi"
mkdir -p "$WORKDIR/temporary"

printf "${DONE_START}Directories have been organized successfully!${DONE_END}"

# Install keep-core contracts abi.

printf "${LOG_START}Installing keep-core contracts ABI...${LOG_END}"

cd "$WORKDIR/temporary"
git clone git@github.com:keep-network/keep-core.git

cd "$WORKDIR/temporary/keep-core/solidity"
npm install

cd "$WORKDIR/temporary/keep-core"
go generate ./...

cd "$WORKDIR"
cp -a "$WORKDIR/temporary/keep-core/pkg/chain/gen/abi/." "$WORKDIR/pkg/chain/gen/core/abi"

printf "${DONE_START}keep-core contracts ABI have been installed successfully!${DONE_END}"


# Create ERC20 abi

printf "${LOG_START}Generating ERC20 contract ABI...${LOG_END}"

cd "$WORKDIR"

solc --abi solidity/ERC20.sol -o "$WORKDIR/pkg/chain/gen/erc20/abi"
abigen --abi="$WORKDIR/pkg/chain/gen/erc20/abi/ERC20.abi" --pkg=token --out="$WORKDIR/pkg/chain/gen/erc20/abi/ERC20.go"

printf "${DONE_START}ERC20 contract ABI have been created successfully!${DONE_END}"

# Cleanup temporary data.

printf "${LOG_START}Cleaning temporary data...${LOG_END}"

rm -rf "$WORKDIR/temporary"

printf "${DONE_START}Temporary data have been cleaned up successfully!${DONE_END}"

# Build the binary.

printf "${LOG_START}Building the binary...${LOG_END}"

go build

printf "${DONE_START}Binary has been built successfully!${DONE_END}"
