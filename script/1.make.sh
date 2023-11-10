#!/bin/bash

docker_id="ketidevit2"
image_name="exascale.metric-collector"
version="latest"

docker build -t $docker_id/$image_name:$version ../build && \
docker push $docker_id/$image_name:$version
