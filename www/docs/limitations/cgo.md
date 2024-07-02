# CGO

Compiling with CGO is a bit trickier.
It won't work "out of the box" with or without GoReleaser: you have to set more
things up.

[This cookbook](../cookbooks/cgo-and-crosscompiling.md) contains more
information.

Tools like `xgo` are not natively supported, and we make no promises about
whether or how well they work within GoReleaser or not.
