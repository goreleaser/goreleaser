# CircleCI

Here is how to do it with [CircleCI](https://circleci.com):

```yml
# .circleci/config.yml
version: 2.1
workflows:
  main:
    jobs:
      - release:
          # Only run this job on git tag pushes
          filters:
            branches:
              ignore: /.*/
            tags:
              only: /v[0-9]+(\.[0-9]+)*(-.*)*/
jobs:
  release:
    docker:
      - image: circleci/golang:1.14
    steps:
      - checkout
      - run: curl -sL https://git.io/goreleaser | bash
```
