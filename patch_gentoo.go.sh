#!/bin/bash
sed -i 's/doinData/installItemData/g' internal/pipe/gentoo/gentoo.go
sed -i 's/doins \[\]doinData/doins \[\]installItemData/g' internal/pipe/gentoo/gentoo.go
