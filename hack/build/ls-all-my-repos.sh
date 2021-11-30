  
#!/bin/bash

# convenient script to find all my repos 
# use with other scripts
# ./hack/build/ls-all-my-repos.sh | xargs -n 1 ./hack/build/check-repo.sh
#  or insanely build all your repos 
 # ./hack/build/ls-all-my-repos.sh | xargs -n 1 ./hack/build/build.sh 

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
if [ -z "$MY_GITHUB_USER" ]
    then
      echo "Missing env var MY_GITHUB_USER, set to your github user and retry."
      exit -1 
fi
curl -s https://api.github.com/users/$MY_GITHUB_USER/repos?per_page=200 | \
    jq -r ".[].html_url" | \
    dos2unix  
