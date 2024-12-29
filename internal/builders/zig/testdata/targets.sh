#!/bin/bash

rm -f ./*targets.txt
zig version >version.txt
zig targets | jq -r '.libc[]' | grep -v freestanding | sort | uniq | while read -r target; do
	# tries to compile a simple hello world in C:
	if zig cc -target "$target" main.c 2>/dev/null; then
		echo "$target" >>./success_targets.txt
		echo "$target" | cut -f1,2 -d- >>./success_targets.txt
	else
		echo "$target" >>./error_targets.txt
		echo "$target" | cut -f1,2 -d- >>./error_targets.txt
	fi
	echo "$target" >>./all_targets.txt
	echo "$target" | cut -f1,2 -d- >>./all_targets.txt
done

sort <./all_targets.txt | uniq >../all_targets.txt
sort <./error_targets.txt | uniq >../error_targets.txt
