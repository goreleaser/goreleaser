#!/bin/bash

rm ./*targets.txt
zig version >version.txt
zig targets | jq -r '.libc[]' | grep -v freestanding | sort | uniq | tee ./all_targets.txt | while read -r target; do
	# tries to compile a simple hello world in C:
	if zig cc -target "$target" main.c 2>/dev/null; then
		echo "$target" >>./success_targets.txt
	else
		echo "$target" >>./error_targets.txt
	fi
done

cp -f ./all_targets.txt ../all_targets.txt
cp -f ./error_targets.txt ../error_targets.txt
