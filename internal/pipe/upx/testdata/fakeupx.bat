@echo off
setlocal enabledelayedexpansion

rem Get the last argument
set args=%*
for %%a in (%*) do set last=%%a

rem Create empty file with .ran extension
type nul > "%last%.ran"

rem Output message
echo Fake compressed: %last%
