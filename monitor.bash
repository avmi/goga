#!/bin/bash

FILES="population.go sorting.go individual.go evolver.go island.go operators.go simplefltprob.go ops_floats.go t_floats_test.go t_sorting_test.go"
#TEST="flt05"
TEST="sort01"

refresh(){
    go test -test.run="$TEST"
}

while true; do
    inotifywait -q -e modify $FILES
    refresh
done
