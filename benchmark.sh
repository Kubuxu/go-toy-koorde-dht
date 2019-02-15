#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

Ns=( 128 256 512 1024 2048 4096 8192 16384 32768)

printf "Nodes, Runs, Forwards, Corrections\n"
for n in ${Ns[@]}; do
	printf '%s, ' $n
	go test -v . -count 1 -run TestResolve/nodes-$n -long | \
		awk '/Looking/{lookup++} /Correcting/{corr++} /Forwarding/{forw++} END{printf "%d, %d, %d\n", lookup+0, forw+0, corr+0}'
done
