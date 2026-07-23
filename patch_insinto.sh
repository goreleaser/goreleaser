#!/bin/bash
sed -i 's/  doins "{{ .Source }}"/  insinto \/\n  doins "{{ .Source }}"/g' internal/pipe/gentoo/gentoo.go
