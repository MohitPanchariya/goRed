#!/bin/bash

num_instances=50

for i in $(seq 1 $num_instances); do
    (
        redis-cli -r 2000 SET "key$i" "value$i"
        redis-cli -r 2000 GET "key$i"
    ) &
done

# Wait for all background processes to finish
wait
