FROM ubuntu:latest
LABEL authors="Administrator"

ENTRYPOINT ["top", "-b"]


# 算咯算咯，没必要整上docker，吃不了细糠