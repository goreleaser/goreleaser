# goreleaser accteptance tests
Currently we test `homebrew-taps` for `gitlab` and `gitea` with a local
setup and some manual steps. To simplify this process we provide as much
automation as possible to set up a local instance.

## gitea
```sh
# starts and initializes a local server
acc-test/local-gitea.sh
# shutdown
docker-compose -f acc-test/docker-compose-gitea.yml down
```

## gitlab
TBD