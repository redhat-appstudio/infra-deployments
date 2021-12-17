#!/bin/bash 

LOG=$(mktemp)
tkn clustertask start \
  cleanup-build-directories \
  --showlog \
  -w name=source,claimName=app-studio-default-workspace | tee $LOG

NAME=$(grep "TaskRun started" $LOG  | cut -d ' ' -f 3)
echo $NAME 
oc delete taskrun $NAME

 