#!/bin/bash

source $(dirname $0)/_helpers.sh

echo ROOT=$ROOT
echo HACK_CHAINS=$HACK_CHAINS
echo say QUIET=$QUIET
echo say FAST=$FAST

title "Hey now"

pause "Hit enter now please..."

title "Yep"

show-then-run date

title "Some yaml"

say "This won't show in quiet mode"
echo "This will show in quiet mode"

echo '{"foo":"bar"}' | yq-pretty
