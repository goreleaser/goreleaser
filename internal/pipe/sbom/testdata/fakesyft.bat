@echo off
setlocal enabledelayedexpansion

rem Parse args to find output files from --output spdx-json=<path> patterns
rem Note: for %%a in (%*) splits on = so we must use shift-based parsing
:loop
if "%~1"=="" goto :done
set "arg=%~1"
if "!arg:~0,10!"=="spdx-json=" (
    set "outfile=!arg:~10!"
    echo {"spdxVersion":"SPDX-2.3","name":"fake"} > "!outfile!"
    echo Fake cataloged: !outfile!
)
shift
goto :loop
:done
