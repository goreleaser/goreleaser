#!/bin/bash

set -o pipefail && TEST_OPTIONS="-json" task test | tee output.json | tparse -follow
success=$?

set -e
NO_COLOR=1 tparse -format markdown -slow 10 -file output.json > $GITHUB_STEP_SUMMARY

exit $success
