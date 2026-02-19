@echo off
setlocal enabledelayedexpansion

rem Parse args to find output files from --output spdx-json=<path> patterns
for %%a in (%*) do (
    set "arg=%%a"
    if "!arg:~0,10!"=="spdx-json=" (
        set "outfile=!arg:~10!"
        echo {"spdxVersion":"SPDX-2.3","name":"fake"} > "!outfile!"
        echo Fake cataloged: !outfile!
    )
)
