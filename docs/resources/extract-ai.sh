#!/bin/bash

FOLDER="${FOLDER:-$(pwd)}"
OCP_RELEASE_LIST="${OCP_RELEASE_LIST:-ai-images.txt}"
BINARY_FOLDER=/var/mnt
CPUS=$(nproc --all)
MAX_CPU_MULT=0.8
MAX_BG=$((jq -n "$CPUS*$MAX_CPU_MULT") | cut -d . -f1)

pushd $FOLDER

load_images() {

  declare -A pids # Hash that include the images pulled along with their pids to be monitored by wait command

  local max_bg=$MAX_BG # Max number of simultaneous skopeo copies to container storage
  local total_copies=$(sort -u $BINARY_FOLDER/$OCP_RELEASE_LIST | wc -l)  # Required to keep track of the pull task vs total
  local current_copy=1

  #remove duplicates
  sort -u -o $OCP_RELEASE_LIST $OCP_RELEASE_LIST
  echo "[INFO] Ready to extract ${total_copies} images using $MAX_BG simultaneous processes"

  while read -r line;
  do
    uri=$(echo "$line" | awk '{print$1}')
    podman image exists $uri
    if [[ $? -eq 0 ]]; then
      echo "[INFO] Skipping existing image $tar"
      echo "[INFO] Copying ${uri} [${current_copy}/${total_copies}]"
      current_copy=$((current_copy + 1))
      continue
    fi
    tar=$(basename ${uri/:/_})
    tar --use-compress-program=pigz -xf ${tar}.tgz
    if [ $? -ne 0 ]; then 
      echo "[ERROR] Could not extract the image ${tar}.gz. Moving to next image" 
      failed_copies+=(${tar}) # Failed, then add the image to be retrieved later
      current_copy=$((current_copy + 1))
      continue 
    fi
    echo "[INFO] Copying ${uri} [${current_copy}/${total_copies}]"
    skopeo copy dir://$(pwd)/${tar} containers-storage:${uri} -q &

    pids[${uri}]=$! # Keeping track of the PID and container image in case the pull fails
    max_bg=$((max_bg - 1)) # Batch size adapted 
    current_copy=$((current_copy + 1)) 
    if [[ $max_bg -eq 0 ]] || [[ $current_copy -gt $total_copies ]] # If the batch is done, then monitor the status of all pulls before moving to the next batch. If the last images are being pulled wait too.
    then
      for img in ${!pids[@]}; do
        wait ${pids[$img]} # The way wait monitor for each background task (PID). If any error then copy the image in the failed array so it can be retried later
        if [[ $? != 0 ]]; then
          echo "[ERROR] Pull failed for container image: ${img} . Retrying later... "
          failed_copies+=(${img}) # Failed, then add the image to be retrieved later
        else
          echo "[INFO] Removing folder for ${img}"
          img_folder=$(basename ${img/:/_})
          rm -rf ${img_folder}
        fi
      done
      # Once the batch is processed, reset the new batch size and clear the processes hash for the next one
      max_bg=$MAX_BG
      pids=()
    fi
  done < ${BINARY_FOLDER}/${OCP_RELEASE_LIST}
}

retry_images() {
  echo "[RETRYING]"
  echo ""
  local rv=0
  for failed_copy in ${failed_copies[@]}; do
    echo "[RETRY] Retrying failed image pull: ${failed_copy}"
    tar=$(basename ${failed_copy/:/_})
    tar --use-compress-program=pigz -xf ${tar}.tgz
    if [ $? -ne 0 ]; then
      echo "[RETRY ERROR] Could not extract the image ${tar}.gz. Moving to next image"
      rv=1
      continue
    fi
    skopeo copy --retry-times 10 dir://$(pwd)/${tar} containers-storage:${failed_copy} -q
    if [[ $? -eq 0 ]]; then  
      rm -rf ${tar}
    else
     echo "[ERROR] Limit number of retries reached. The image could not be pulled: ${failed_copy}"
     rv=1
    fi
  done
  echo "[INFO] Image load done"
  return $rv
}

if [[ "${BASH_SOURCE[0]}" = "${0}" ]]; then
  failed_copies=() # Array that will include all the images that failed to be pulled
  load_images
  retry_images # Return 1 if max.retries reached
  if [[ $? -ne 0 ]]; then
    echo "[FAIL] ${#failed_copies[@]} images were not precached successfully" #number of failing images
    exit 1
  else
    echo "[SUCCESS] All images were precached"
    exit 0
  fi
fi
