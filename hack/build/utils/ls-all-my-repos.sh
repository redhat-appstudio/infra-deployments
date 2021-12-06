  
#!/bin/bash

# convenient script to find all my repos
# require an env var MY_GITHUB_USER
# Combine with other scripts
# List all the repos you own and determine build type 
# ./hack/build/utils/ls-all-my-repos.sh | xargs -n 1 ./hack/build/utils/check-repo.sh
# Insanely build all the repos 
 # ./hack/build/utils/ls-all-my-repos.sh | xargs -n 1 ./hack/build/build.sh 
 
if [ -z "$MY_GITHUB_USER" ]
    then
      echo "Missing env var MY_GITHUB_USER, set to your github user and retry."
      exit -1 
fi
curl -s https://api.github.com/users/$MY_GITHUB_USER/repos?per_page=200 | \
    jq -r ".[].html_url" | \
    dos2unix  
