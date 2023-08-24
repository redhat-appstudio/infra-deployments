#!/bin/bash
#This script sets up the pre-requisites for setting up app-studio in local machine.
# mainly setting up pip3,yq, kubectl, kustomize
# Firstly it checks which type of system is running this
# Then it installs the dependencies one by one

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
      *"Red Hat Enterprise Linux"*)
        echo "I am in RHEL"
        $ROOT/redhat-based.sh
        ;;

      *"Fedora"*)
        echo "I am in Fedora"
        $ROOT/redhat-based.sh
        ;;

      *"Ubuntu"*)
        echo "I am in Ubuntu"
        $ROOT/debian-based.sh
        ;;

      *)
        echo "OS not supported by this script"
        ;;
    esac
    ;;
  *)
    echo "OS not supported by this script"
    ;;
esac
