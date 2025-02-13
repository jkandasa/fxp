#!/bin/bash

BUILDS_DIR=builds
mkdir -p ${BUILDS_DIR}
rm ${BUILDS_DIR} -rf

# platforms to build
PLATFORMS=("linux/arm64" "linux/amd64" "darwin/amd64" "windows/amd64")

# compile
for platform in "${PLATFORMS[@]}"
do
  platform_raw=(${platform//\// })
  GOOS=${platform_raw[0]}
  GOARCH=${platform_raw[1]}
  package_name="fxp-${GOOS}-${GOARCH}"

  env GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 go build -o ${BUILDS_DIR}/${package_name} main.go
  if [ $? -ne 0 ]; then
    echo 'an error has occurred. aborting the build process'
    exit 1
  fi
done
