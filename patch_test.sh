#!/bin/bash
sed -i 's/config.Doin/config.GentooInstallItem/g' internal/pipe/gentoo/gentoo_test.go
sed -i 's/Doins \[\]doinData/Doins \[\]installItemData/g' internal/pipe/gentoo/gentoo_test.go
