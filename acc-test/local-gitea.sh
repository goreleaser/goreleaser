#!/bin/bash

set -e

WORK_DIR=$(pwd)
USER="goreleaser"
PASSWORD="testpwd123!"
EMAIL="goreleaser@acme.com"
SSH_KEY="$HOME/.ssh/id_rsa_goreleaser"
HOST="localhost"
PORT="3000"
CONTAINER_NAME="local-gitea"

# 0 check prerequisites
# docker-compose, nc, ssh-keygen

rm -rf "$WORK_DIR"/acc-test/gitea

# 0 check for local ssh key - create if necessary
if [ ! -f "$SSH_KEY" ]; then
 echo "SSH KEY at $SSH_KEY does not exist. Creating"
 ssh-keygen -t rsa -b 4096 -C "$EMAIL" -f "$SSH_KEY" -q -N ""
fi
# add it to the agent
ssh-add "$SSH_KEY"
ssh-add -l

# 1 start gitea
docker-compose -f "$WORK_DIR"/acc-test/docker-compose-gitea.yml up -d

# 2 wait until up
while ! $(nc -z -v -w5 $HOST $PORT); do
    echo "Waiting for '$HOST/$PORT' to come up... sleep 2"
    sleep 2
done
echo "'$HOST/$PORT' is up. Continuing."
sleep 5

# 3 create goreleaser user
# this command creates the user and an access token as well
ACCESS_TOKEN=$(docker exec $CONTAINER_NAME \
    gitea admin user create \
        --username=${USER} \
        --password=${PASSWORD} \
        --email=${EMAIL} \
        --admin=true \
        --must-change-password=false \
        --access-token \
        | grep -o "Access token was successfully created....*" \
        | sed -ne 's/^.*Access token was successfully created... //p')

echo "ACCESS_TOKEN: '$ACCESS_TOKEN'"
if [ -z "$ACCESS_TOKEN" ]; then
    echo "Empty ACCESS_TOKEN"
    exit 1
fi

# add the pub key
# the 'gitea admin user create' command does not have the option to do this
PAYLOAD=$(echo ' 
{ 
    "title": "'${USER}'",
    "key": "'$(cat ${SSH_KEY}.pub)'",
    "read_only": false}
}
')
curl -f -X POST "http://$HOST:$PORT/api/v1/user/keys" \
    -H "accept: application/json" \
    -H "Authorization: token $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD" 

# 4 create gorleaser-testing repo
PAYLOAD=$(echo ' 
{
    "auto_init": false,
    "name": "goreleaser-testing"
}
')
curl -f -X POST "http://$HOST:$PORT/api/v1/user/repos" \
    -H "accept: application/json" \
    -H "Authorization: token $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD"


# 5 create homebrew-tap repo
PAYLOAD=$(echo ' 
{
    "auto_init": true,
    "name": "homebrew-tap"
}
')
curl -f -X POST "http://$HOST:$PORT/api/v1/user/repos" \
    -H "accept: application/json" \
    -H "Authorization: token $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD"

# # create readme.file
# README=$(echo "
# # homebrew tap
# ## Usage
# \`\`\`sh
# brew tap ${USER}/gitea-tools ssh://git@localhost:222/${USER}/homebrew-tap.git
# brew install ${USER}/gitea-tools/goreleaser
# \`\`\`
# "
# )

# create the Formula path
# PAYLOAD=$(echo ' 
# {
#   "content": "",
#   "message": "chore: adds Formula folder"
# }
# ')
# curl -f -X POST "http://$HOST:$PORT/api/v1/repos/${USER}/homebrew-tap/contents/Formula/.gitkeep" \
#     -H "accept: application/json" \
#     -H "Authorization: token $ACCESS_TOKEN" \
#     -H "Content-Type: application/json" \
#     -d "$PAYLOAD"

# 6 push to gitea repo
# fails atm: https://github.com/wkulhanek/docker-openshift-gitea/issues/9
git remote add origin ssh://git@localhost:222/goreleaser/goreleaser-testing.git
ssh -vvvT git@localhost -p 222
