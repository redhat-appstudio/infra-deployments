#!/bin/bash
# This script sets up the pre-requisites for running the chains demos.
# Firstly it checks which type of system is running this
# Then it installs the dependencies one by one
#
# Todo: Integrate this into hack/setup since there's some overlap
#

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
echo $ROOT
#TODO: check the ROOT directory

case $OSTYPE in
  *"darwin"*)
    echo "I am in Macos"
    $ROOT/macos-based.sh
    ;;
  *"linux"*)
    OSNAME=$(grep '^NAME=' /etc/os-release)
    echo $OSNAME

    case $OSNAME in
      *"Fedora"*)
        echo "I am in Fedora"
        $ROOT/redhat-based.sh
        ;;

      *"Ubuntu"*)
        echo "I am in Ubuntu"
        $ROOT/debian-based.sh
        ;;
    esac
    ;;
esac
